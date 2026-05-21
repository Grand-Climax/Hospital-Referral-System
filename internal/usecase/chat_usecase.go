package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	irepository "Hospital-Referral-System/internal/domain/interfaces/repository"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
	"Hospital-Referral-System/internal/infrastructure/ws"
)

type chatUseCase struct {
	chatRepo           irepository.ChatMessageRepository
	referralRepo       irepository.ReferralRepository
	userRepo           irepository.UserRepository
	referralAccessRepo irepository.ReferralAccessRepository
	netRepo            irepository.NetworkRepository
	db                 *gorm.DB
	hub                *ws.Hub
	auditRepo          irepository.AuditLogRepository
}

func NewChatUseCase(
	chatRepo irepository.ChatMessageRepository,
	referralRepo irepository.ReferralRepository,
	userRepo irepository.UserRepository,
	referralAccessRepo irepository.ReferralAccessRepository,
	netRepo irepository.NetworkRepository,
	db *gorm.DB,
	hub *ws.Hub,
	auditRepo irepository.AuditLogRepository,
) iusecase.ChatUseCase {
	return &chatUseCase{
		chatRepo:           chatRepo,
		referralRepo:       referralRepo,
		userRepo:           userRepo,
		referralAccessRepo: referralAccessRepo,
		netRepo:            netRepo,
		db:                 db,
		hub:                hub,
		auditRepo:          auditRepo,
	}
}

func (u *chatUseCase) SendMessage(ctx context.Context, senderID, receiverID uuid.UUID, referralID *uuid.UUID, content string) (*entity.ChatMessage, error) {
	// 1. Validate sender ≠ receiver
	if senderID == receiverID {
		return nil, errors.New("Cannot send a message to yourself.")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("Content is required.")
	}
	if len(content) > 5000 {
		return nil, errors.New("Content must not exceed 5000 characters.")
	}

	// 2. Fetch both users
	sender, err := u.userRepo.FindByID(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sender: %v", err)
	}
	receiver, err := u.userRepo.FindByID(ctx, receiverID)
	if err != nil {
		return nil, errors.New("Recipient account is inactive.")
	}

	// Verify both users are active
	if !sender.IsActive || !receiver.IsActive || sender.IsDeleted || receiver.IsDeleted {
		return nil, errors.New("Recipient account is inactive.")
	}

	// Verify MOH_ANALYST cannot participate in chat
	if sender.Role == entity.RoleMohAnalyst {
		return nil, errors.New("MOH Analysts cannot use the chat system.")
	}
	if receiver.Role == entity.RoleMohAnalyst {
		return nil, errors.New("MOH Analysts cannot receive chat messages.")
	}

	// General receptionist boundary check
	if receiver.Role == entity.RoleReceptionist {
		if sender.Role != entity.RoleReferringDoctor && sender.Role != entity.RoleReceivingSpecialist {
			return nil, errors.New("Receptionists can only receive messages from clinical staff in their hospital.")
		}
		if sender.HospitalID == nil || receiver.HospitalID == nil || *sender.HospitalID != *receiver.HospitalID {
			return nil, errors.New("Receptionists can only receive messages from clinical staff in their hospital.")
		}
	}

	// 3. Check reply exception (existing conversation?)
	hasPriorConv, err := u.chatRepo.HasConversation(ctx, senderID, receiverID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify chat history: %v", err)
	}

	var conv *entity.Conversation

	// 4. If referralID != nil → validate referral exists, not terminal/closed, connected
	if referralID != nil {
		referral, err := u.referralRepo.GetReferralByID(ctx, *referralID)
		if err != nil {
			return nil, errors.New("Referral not found.")
		}

		// Terminal referral checks (read-only)
		if referral.Status == entity.StatusCompleted ||
			referral.Status == entity.StatusDeceased ||
			referral.Status == entity.StatusCancelled ||
			referral.Status == entity.StatusRejectedByLiaison ||
			referral.Status == entity.StatusRejectedBySpecialist ||
			referral.Status == entity.StatusRejectedAfterSend ||
			referral.Status == entity.StatusRedirected {
			return nil, errors.New("This referral has been completed. The conversation is now read‑only.")
		}

		// Retrieve or create referral-scoped conversation associated with the current TargetHospitalID
		conv, err = u.chatRepo.GetOrCreateReferralConversation(ctx, *referralID, referral.TargetHospitalID)
		if err != nil {
			return nil, fmt.Errorf("failed to load referral conversation: %v", err)
		}

		// Enforce referral connection rules if no prior chat history existed
		if !hasPriorConv {
			senderConnected, err := u.isUserConnectedToReferral(ctx, senderID, sender, referral)
			if err != nil {
				return nil, err
			}
			if !senderConnected {
				return nil, errors.New("You are not authorised to participate in this referral's conversation.")
			}

			receiverConnected, err := u.isUserConnectedToReferral(ctx, receiverID, receiver, referral)
			if err != nil {
				return nil, err
			}
			if !receiverConnected {
				return nil, errors.New("The recipient is not connected to this referral.")
			}
		}
	} else {
		// 5. If referralID == nil → enforce initiation matrix if no prior history
		if !hasPriorConv {
			if err := u.enforceInitiationMatrix(ctx, sender, receiver); err != nil {
				return nil, err
			}
		}

		// Get or create unique direct conversation
		conv, err = u.chatRepo.GetOrCreateDirectConversation(ctx, senderID, receiverID)
		if err != nil {
			return nil, fmt.Errorf("failed to load direct conversation: %v", err)
		}
	}

	// 6. Assert conversation is not administrative-disabled
	if conv.IsDisabled {
		return nil, errors.New("This chat room has been administrative-locked.")
	}

	// Ensure both users are registered as participants in this conversation
	_ = u.chatRepo.EnsureParticipant(ctx, conv.ID, senderID)
	_ = u.chatRepo.EnsureParticipant(ctx, conv.ID, receiverID)

	// 7. Save message under this conversation
	chatMsg := &entity.ChatMessage{
		ID:             uuid.New(),
		ConversationID: conv.ID,
		SenderID:       senderID,
		Content:        content,
		CreatedAt:      time.Now(),
	}

	if err := u.chatRepo.Create(ctx, chatMsg); err != nil {
		return nil, fmt.Errorf("failed to save message: %v", err)
	}

	// 8. Push via WebSocket
	if u.hub != nil {
		var refIDStr *string
		if conv.ReferralID != nil {
			s := conv.ReferralID.String()
			refIDStr = &s
		}
		wsMsg := dto.WebSocketMessage{
			Type: "chat",
			Data: dto.ChatMessageResponse{
				ID:             chatMsg.ID.String(),
				ConversationID: conv.ID.String(),
				ReferralID:     refIDStr,
				SenderID:       chatMsg.SenderID.String(),
				ReceiverID:     receiverID.String(),
				Content:        chatMsg.Content,
				CreatedAt:      chatMsg.CreatedAt.Format(time.RFC3339),
			},
		}
		go u.hub.SendToUser(receiverID.String(), wsMsg)
	}

	return chatMsg, nil
}

func (u *chatUseCase) ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int64, error) {
	conversations, total, err := u.chatRepo.GetConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var responses []dto.ConversationResponse
	for _, cm := range conversations {
		var otherUser *entity.User
		for _, p := range cm.Participants {
			if p.UserID != userID && p.User != nil {
				otherUser = p.User
				break
			}
		}

		// Stakeholder fallback for referral rooms
		if otherUser == nil && cm.Referral != nil {
			var otherUserID uuid.UUID
			if cm.Referral.ReferringDoctorID != userID {
				otherUserID = cm.Referral.ReferringDoctorID
			} else if cm.Referral.SpecialistID != nil && *cm.Referral.SpecialistID != userID {
				otherUserID = *cm.Referral.SpecialistID
			} else if cm.Referral.LiaisonOfficerID != nil && *cm.Referral.LiaisonOfficerID != userID {
				otherUserID = *cm.Referral.LiaisonOfficerID
			}

			if otherUserID != uuid.Nil {
				otherUser, _ = u.userRepo.FindByID(ctx, otherUserID)
			}
		}

		if otherUser == nil {
			continue
		}

		var hospitalName string
		if otherUser.Hospital != nil {
			hospitalName = otherUser.Hospital.Name
		} else {
			hospitalName = "System"
		}

		unreadCount, _ := u.chatRepo.GetConversationUnreadCount(ctx, cm.ID, userID)

		isReadOnly := false
		if cm.ReferralID != nil && cm.Referral != nil {
			if cm.Referral.Status == entity.StatusCompleted ||
				cm.Referral.Status == entity.StatusDeceased ||
				cm.Referral.Status == entity.StatusCancelled ||
				cm.Referral.Status == entity.StatusRejectedByLiaison ||
				cm.Referral.Status == entity.StatusRejectedBySpecialist ||
				cm.Referral.Status == entity.StatusRejectedAfterSend ||
				cm.Referral.Status == entity.StatusRedirected {
				isReadOnly = true
			}
		}

		var refIDStr *string
		if cm.ReferralID != nil {
			s := cm.ReferralID.String()
			refIDStr = &s
		}

		preview := cm.LastMessageContent
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}

		responses = append(responses, dto.ConversationResponse{
			ConversationID:    cm.ID.String(),
			OtherUserID:       otherUser.ID.String(),
			OtherUserName:     fmt.Sprintf("%s %s", otherUser.FirstName, otherUser.LastName),
			OtherUserRole:     string(otherUser.Role),
			OtherUserHospital: hospitalName,
			LastMessage:       preview,
			LastMessageAt:     cm.LastMessageAt.Format(time.RFC3339),
			UnreadCount:       unreadCount,
			ReferralID:        refIDStr,
			IsReadOnly:        isReadOnly,
			IsDisabled:        cm.IsDisabled,
			DisabledReason:    cm.DisabledReason,
		})
	}

	return responses, total, nil
}

func (u *chatUseCase) GetMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, limit, offset int) ([]entity.ChatMessage, int64, error) {
	conv, err := u.chatRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, 0, errors.New("Conversation not found.")
	}

	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, 0, errors.New("User not found.")
	}

	// Access control check (Spyping / Administrative Audits support)
	isAuthorized := false
	if user.Role == entity.RoleSystemSuperAdmin {
		isAuthorized = true
	} else if user.Role == entity.RoleHospitalAdmin && user.HospitalID != nil {
		if conv.Referral != nil && (conv.Referral.SenderHospitalID == *user.HospitalID || conv.Referral.TargetHospitalID == *user.HospitalID) {
			isAuthorized = true
		} else {
			for _, p := range conv.Participants {
				if p.User != nil && p.User.HospitalID != nil && *p.User.HospitalID == *user.HospitalID {
					isAuthorized = true
					break
				}
			}
		}
	} else {
		for _, p := range conv.Participants {
			if p.UserID == userID {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return nil, 0, errors.New("You are not authorised to view this conversation.")
	}

	messages, total, err := u.chatRepo.GetMessages(ctx, conversationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Mark all read
	if err := u.chatRepo.MarkRead(ctx, conversationID, userID); err != nil {
		fmt.Printf("Failed to mark messages as read: %v\n", err)
	}

	return messages, total, nil
}

func (u *chatUseCase) MarkRead(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	return u.chatRepo.MarkRead(ctx, conversationID, userID)
}

func (u *chatUseCase) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return u.chatRepo.GetUnreadCount(ctx, userID)
}

func (u *chatUseCase) ToggleDisabled(ctx context.Context, adminID uuid.UUID, conversationID uuid.UUID, isDisabled bool, reason string) error {
	admin, err := u.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return errors.New("Admin account not found.")
	}

	conv, err := u.chatRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return errors.New("Conversation not found.")
	}

	// Hospital admins can only disable chats involving their hospital
	if admin.Role == entity.RoleHospitalAdmin && admin.HospitalID != nil {
		isAllowed := false
		if conv.Referral != nil && (conv.Referral.SenderHospitalID == *admin.HospitalID || conv.Referral.TargetHospitalID == *admin.HospitalID) {
			isAllowed = true
		} else {
			for _, p := range conv.Participants {
				if p.User != nil && p.User.HospitalID != nil && *p.User.HospitalID == *admin.HospitalID {
					isAllowed = true
					break
				}
			}
		}
		if !isAllowed {
			return errors.New("unauthorised: hospital admins can only lock chats within their hospital boundary")
		}
	} else if admin.Role != entity.RoleSystemSuperAdmin {
		return errors.New("unauthorised: only admins can administrative-lock chats")
	}

	// Save updates
	if err := u.chatRepo.ToggleDisabled(ctx, conversationID, isDisabled, reason, &adminID); err != nil {
		return err
	}

	// Write System Audit Log
	oldState := "enabled"
	newState := "disabled"
	if !isDisabled {
		oldState = "disabled"
		newState = "enabled"
	}
	_ = u.auditRepo.LogWithContext(ctx, adminID, entity.ActionManageChat, conv.ReferralID, oldState, newState)

	return nil
}

func (u *chatUseCase) SoftDeleteConversation(ctx context.Context, adminID uuid.UUID, conversationID uuid.UUID) error {
	admin, err := u.userRepo.FindByID(ctx, adminID)
	if err != nil {
		return errors.New("Admin account not found.")
	}

	conv, err := u.chatRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return errors.New("Conversation not found.")
	}

	// Role validations
	if admin.Role == entity.RoleHospitalAdmin && admin.HospitalID != nil {
		isAllowed := false
		if conv.Referral != nil && (conv.Referral.SenderHospitalID == *admin.HospitalID || conv.Referral.TargetHospitalID == *admin.HospitalID) {
			isAllowed = true
		} else {
			for _, p := range conv.Participants {
				if p.User != nil && p.User.HospitalID != nil && *p.User.HospitalID == *admin.HospitalID {
					isAllowed = true
					break
				}
			}
		}
		if !isAllowed {
			return errors.New("unauthorised: hospital admins can only delete chats within their hospital boundary")
		}
	} else if admin.Role != entity.RoleSystemSuperAdmin {
		return errors.New("unauthorised: only admins can delete conversations")
	}

	if err := u.chatRepo.SoftDeleteConversation(ctx, conversationID); err != nil {
		return err
	}

	// Write System Audit Log
	_ = u.auditRepo.LogWithContext(ctx, adminID, entity.ActionManageChat, conv.ReferralID, "active", "soft_deleted")

	return nil
}

func (u *chatUseCase) isUserConnectedToReferral(ctx context.Context, userID uuid.UUID, user *entity.User, referral *entity.Referral) (bool, error) {
	if user.Role == entity.RoleSystemSuperAdmin {
		return true, nil
	}

	if referral.ReferringDoctorID == userID {
		return true, nil
	}

	if referral.LiaisonOfficerID != nil && *referral.LiaisonOfficerID == userID {
		return true, nil
	}

	if referral.SpecialistID != nil && *referral.SpecialistID == userID {
		return true, nil
	}

	if user.Role == entity.RoleHospitalAdmin && user.HospitalID != nil {
		if referral.SenderHospitalID == *user.HospitalID || referral.TargetHospitalID == *user.HospitalID {
			return true, nil
		}
	}

	if user.Role == entity.RoleDeptHead && user.HospitalID != nil && user.DepartmentID != nil {
		if referral.TargetHospitalID == *user.HospitalID && referral.TargetDeptID == *user.DepartmentID {
			return true, nil
		}
	}

	// Active ReferralAccess
	hasAccess, err := u.referralAccessRepo.CheckAccess(ctx, referral.ID, userID)
	if err != nil {
		return false, err
	}
	if hasAccess {
		return true, nil
	}

	return false, nil
}

func (u *chatUseCase) enforceInitiationMatrix(ctx context.Context, sender, receiver *entity.User) error {
	if u.db == nil {
		return nil
	}
	senderHospitalID := uuid.Nil
	if sender.HospitalID != nil {
		senderHospitalID = *sender.HospitalID
	}
	receiverHospitalID := uuid.Nil
	if receiver.HospitalID != nil {
		receiverHospitalID = *receiver.HospitalID
	}

	switch sender.Role {
	case entity.RoleReferringDoctor:
		switch receiver.Role {
		case entity.RoleReferringDoctor:
			if senderHospitalID == uuid.Nil || senderHospitalID != receiverHospitalID {
				return errors.New("You can only chat with colleagues in your own hospital.")
			}
			return nil

		case entity.RoleReceivingSpecialist:
			var accessCount int64
			err := u.db.Model(&entity.ReferralAccess{}).
				Joins("JOIN referrals ON referrals.id = referral_accesses.referral_id").
				Where("referrals.referring_doctor_id = ? AND referral_accesses.user_id = ? AND referral_accesses.revoked_at IS NULL", sender.ID, receiver.ID).
				Count(&accessCount).Error
			if err != nil {
				return err
			}
			if accessCount > 0 {
				return nil
			}

			var refCount int64
			err = u.db.Model(&entity.Referral{}).
				Where("referring_doctor_id = ? AND specialist_id = ? AND status IN (?)", sender.ID, receiver.ID, []entity.ReferralStatus{entity.StatusAccepted, entity.StatusScheduled, entity.StatusCompleted}).
				Count(&refCount).Error
			if err != nil {
				return err
			}
			if refCount > 0 {
				return nil
			}

			return errors.New("You can only chat with specialists who have active access or accepted your referrals.")

		case entity.RoleLiaisonOfficer:
			if receiverHospitalID == uuid.Nil {
				return errors.New("Liaison hospital is not configured.")
			}
			var count int64
			err := u.db.Model(&entity.Referral{}).
				Where("referring_doctor_id = ? AND sender_hospital_id = ?", sender.ID, receiverHospitalID).
				Count(&count).Error
			if err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
			return errors.New("A referral must exist at the liaison's hospital to initiate chat.")

		case entity.RoleReceptionist:
			if senderHospitalID == uuid.Nil || senderHospitalID != receiverHospitalID {
				return errors.New("You can only chat with colleagues in your own hospital.")
			}
			return nil

		case entity.RoleDeptHead:
			if senderHospitalID == uuid.Nil || senderHospitalID != receiverHospitalID {
				return errors.New("You can only chat with colleagues in your own hospital.")
			}
			return nil
		}

	case entity.RoleReceivingSpecialist:
		switch receiver.Role {
		case entity.RoleReferringDoctor:
			var accessCount int64
			err := u.db.Model(&entity.ReferralAccess{}).
				Joins("JOIN referrals ON referrals.id = referral_accesses.referral_id").
				Where("referrals.referring_doctor_id = ? AND referral_accesses.user_id = ? AND referral_accesses.revoked_at IS NULL", receiver.ID, sender.ID).
				Count(&accessCount).Error
			if err != nil {
				return err
			}
			if accessCount > 0 {
				return nil
			}

			var refCount int64
			err = u.db.Model(&entity.Referral{}).
				Where("referring_doctor_id = ? AND specialist_id = ? AND status IN (?)", receiver.ID, sender.ID, []entity.ReferralStatus{entity.StatusAccepted, entity.StatusScheduled, entity.StatusCompleted}).
				Count(&refCount).Error
			if err != nil {
				return err
			}
			if refCount > 0 {
				return nil
			}

			return errors.New("You can only initiate chat with a doctor if you have active access to their referrals.")

		case entity.RoleReceivingSpecialist:
			if senderHospitalID == uuid.Nil || senderHospitalID != receiverHospitalID {
				return errors.New("You can only chat with colleagues in your own hospital.")
			}
			return nil
		}

	case entity.RoleLiaisonOfficer:
		switch receiver.Role {
		case entity.RoleReferringDoctor:
			if senderHospitalID == uuid.Nil {
				return errors.New("Your hospital is not configured.")
			}
			var count int64
			err := u.db.Model(&entity.Referral{}).
				Where("referring_doctor_id = ? AND sender_hospital_id = ?", receiver.ID, senderHospitalID).
				Count(&count).Error
			if err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
			return errors.New("A referral must exist to initiate chat with this doctor.")

		case entity.RoleReceivingSpecialist:
			if senderHospitalID == uuid.Nil || receiverHospitalID == uuid.Nil {
				return errors.New("Hospitals are not configured.")
			}
			routeExists1, err := u.netRepo.VerifyNetworkPathway(ctx, senderHospitalID, receiverHospitalID)
			if err != nil {
				return err
			}
			routeExists2, err := u.netRepo.VerifyNetworkPathway(ctx, receiverHospitalID, senderHospitalID)
			if err != nil {
				return err
			}
			if routeExists1 || routeExists2 {
				return nil
			}
			return errors.New("No network route exists between your hospital and the specialist's hospital.")
		}

	case entity.RoleHospitalAdmin:
		if receiver.Role != entity.RoleMohAnalyst && receiver.Role != entity.RoleSystemSuperAdmin {
			if senderHospitalID != uuid.Nil && senderHospitalID == receiverHospitalID {
				return nil
			}
			return errors.New("Hospital admins can only chat with users in their own hospital.")
		}

	case entity.RoleSystemSuperAdmin:
		if receiver.Role != entity.RoleMohAnalyst {
			return nil
		}

	case entity.RoleReceptionist:
		return errors.New("Receptionists cannot initiate a conversation.")
	}

	return errors.New("You are not authorised to initiate this conversation.")
}

func (u *chatUseCase) GetConversationID(ctx context.Context, userID, otherUserID uuid.UUID, referralID *uuid.UUID) (uuid.UUID, error) {
	if referralID != nil {
		referral, err := u.referralRepo.GetReferralByID(ctx, *referralID)
		if err != nil {
			return uuid.Nil, errors.New("Referral not found.")
		}
		conv, err := u.chatRepo.GetOrCreateReferralConversation(ctx, *referralID, referral.TargetHospitalID)
		if err != nil {
			return uuid.Nil, err
		}
		return conv.ID, nil
	}
	conv, err := u.chatRepo.GetOrCreateDirectConversation(ctx, userID, otherUserID)
	if err != nil {
		return uuid.Nil, err
	}
	return conv.ID, nil
}
