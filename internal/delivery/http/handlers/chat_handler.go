package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/delivery/http/dto"
	"Hospital-Referral-System/internal/domain/entity"
	iusecase "Hospital-Referral-System/internal/domain/interfaces/usecase"
)

type ChatHandler struct {
	chatUC iusecase.ChatUseCase
}

func NewChatHandler(chatUC iusecase.ChatUseCase) *ChatHandler {
	return &ChatHandler{chatUC: chatUC}
}

// SendMessage godoc
// @Summary      Send Chat Message
// @Description  Send a real-time message to another user, either as a direct peer-to-peer message or scoped to a specific clinical referral context.
// @Description  
// @Description  ### Security & Role Authorization Matrix:
// @Description  - **Allowed Roles:** `SYSTEM_SUPER_ADMIN`, `HOSPITAL_ADMIN`, `REFERRING_DOCTOR`, `RECEIVING_SPECIALIST`, `LIAISON_OFFICER`, `RECEPTIONIST`, `DEPT_HEAD`.
// @Description  - **Barred Roles:** `MOH_ANALYST` is strictly forbidden from participating in or receiving chat messages.
// @Description  - **Receptionist Boundaries:** Receptionists cannot initiate peer chats, and can only receive messages from clinical staff (Doctors/Specialists) inside their own hospital.
// @Description  
// @Description  ### Scoped/Referral Chat Restrictions:
// @Description  - **Access Control:** If `referral_id` is supplied, both sender and receiver must be connected to the referral (creator, treating/consulting doctor, department head, liaison, or user with active access grant).
// @Description  - **Terminal States (Read-Only):** The chat room becomes strictly read-only if the referral status is terminal (`COMPLETED`, `DECEASED`, `CANCELLED`, `REJECTED_BY_LIAISON`, `REJECTED_BY_SPECIALIST`, `REJECTED_AFTER_SEND`, `REDIRECTED`).
// @Description  
// @Description  ### Direct (Non-Referral) Peer Chat Initiation:
// @Description  - **Referring Doctor:** Can initiate with own hospital colleagues, liaison officer at a hospital with active referral, or specialists with active access/accepted referrals.
// @Description  - **Liaison Officer:** Can initiate with referring doctors of active referrals, or specialists if a network route exists between their hospitals.
// @Description  - **Receiving Specialist:** Can only initiate if they have active access to the doctor's referrals.
// @Description  - **Hospital Admin:** Can only initiate with staff of their own hospital.
// @Description  
// @Description  ### Prerequisites:
// @Description  - Receiver account must be active and not soft-deleted.
// @Description  - Content must be non-empty and between 1 and 5000 characters.
// @Description  - Sender and receiver must not be the same user.
// @Description  - Conversation must not be administrative-locked.
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        message  body  dto.SendMessageRequest  true  "Message details including receiver, optional referral context, and content text"
// @Success      201  {object}  dto.ChatMessageResponse
// @Failure      400  {object}  dto.ErrorResponse  "Invalid parameters, empty content, content > 5000 chars, self-send, or inactive recipient"
// @Failure      403  {object}  dto.ErrorResponse  "Unauthorized role, multi-hospital boundary violation, terminal referral (read-only), or admin locked"
// @Failure      404  {object}  dto.ErrorResponse  "Referral not found"
// @Failure      500  {object}  dto.ErrorResponse  "Internal database or transmission error"
// @Security     BearerAuth
// @Router       /api/v1/chat/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	msg, err := h.chatUC.SendMessage(c.Request.Context(), userID, req.ReceiverID, req.ReferralID, req.Content)
	if err != nil {
		status := http.StatusForbidden
		errMsg := err.Error()
		if errMsg == "Cannot send a message to yourself." || errMsg == "Recipient account is inactive." || errMsg == "Content is required." || errMsg == "Content must not exceed 5000 characters." {
			status = http.StatusBadRequest
		} else if errMsg == "Referral not found." {
			status = http.StatusNotFound
		}
		c.JSON(status, dto.ErrorResponse{Success: false, Error: errMsg})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessPayload(dto.ToChatMessageResponse(*msg, req.ReceiverID.String()), "Message sent successfully"))
}

// ListConversations godoc
// @Summary      List My Conversations
// @Description  Retrieve a paginated list of all active direct and referral-scoped chat channels involving the authenticated user.
// @Description  
// @Description  ### Features & Fields:
// @Description  - Shows the latest message preview, unread message count for the current user, and other participant metadata.
// @Description  - **`is_read_only` (bool):** Set to `true` if the channel is referral-scoped and has reached a terminal state.
// @Description  - **`is_disabled` (bool):** Set to `true` if an administrator has locked the channel.
// @Description  
// @Description  ### Roles:
// @Description  - Accessible by all authenticated roles except `MOH_ANALYST`.
// @Tags         Chat
// @Produce      json
// @Param        limit  query  int     false  "Pagination limit (safe sanitized minimum of 1)"  default(20)
// @Param        page   query  int     false  "Page number (safe sanitized minimum of 1)"       default(1)
// @Param        type   query  string  false  "Filter by conversation type: all (default), direct (no referral), referral (scoped to a referral)"
// @Success      200  {object}  dto.PaginatedConversationResponse
// @Failure      401  {object}  dto.ErrorResponse  "Unauthorized: Invalid session token"
// @Failure      500  {object}  dto.ErrorResponse  "Internal database retrieval error"
// @Security     BearerAuth
// @Router       /api/v1/chat/conversations [get]
func (h *ChatHandler) ListConversations(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	filterType := c.DefaultQuery("type", "all")
	if filterType != "direct" && filterType != "referral" {
		filterType = "all"
	}

	conversations, total, err := h.chatUC.ListConversations(c.Request.Context(), userID, filterType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedConversationResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Conversations retrieved successfully",
		},
		Data:     conversations,
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}

// GetMessages godoc
// @Summary      Get Message History & Auto-Read
// @Description  Retrieve the message logs for a conversation. If conversation ID is not provided, the system automatically resolves or creates the unique channel using `other_user_id` and optional `referral_id`.
// @Description  
// @Description  ### Security & Administrative Auditing (Spying):
// @Description  - **Normal Participants:** Allowed to view conversations they are registered members of.
// @Description  - **Hospital Admins:** Authorized to view/audit message history if the chat involves staff of their hospital or a referral linked to their hospital.
// @Description  - **System Super Admins:** Unrestricted access to view any conversation globally.
// @Description  - **Other Roles:** Third-party access is rejected with `403 Forbidden`.
// @Description  
// @Description  ### Auto-Read side-effect:
// @Description  - Querying messages automatically marks all unread messages in that channel as read for the calling user, updating badge states.
// @Tags         Chat
// @Produce      json
// @Param        conversation_id query  string  false  "Conversation UUID to load history directly"
// @Param        other_user_id   query  string  false  "Recipient UUID to resolve direct or referral channel"
// @Param        referral_id     query  string  false  "Referral UUID to scope the conversation search"
// @Param        limit           query  int     false  "Pagination limit"  default(20)
// @Param        page            query  int     false  "Page number"       default(1)
// @Success      200  {object}  dto.PaginatedChatResponse
// @Failure      400  {object}  dto.ErrorResponse  "Missing both conversation_id and other_user_id, or malformed UUIDs"
// @Failure      403  {object}  dto.ErrorResponse  "Unauthorized: Boundary check violation or non-participant access"
// @Failure      500  {object}  dto.ErrorResponse  "Internal data processing failure"
// @Security     BearerAuth
// @Router       /api/v1/chat/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	var convID uuid.UUID
	var err error

	if convIDStr := c.Query("conversation_id"); convIDStr != "" {
		convID, err = uuid.Parse(convIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid conversation_id"})
			return
		}
	} else {
		otherUserIDStr := c.Query("other_user_id")
		if otherUserIDStr == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "either conversation_id or other_user_id is required"})
			return
		}

		otherUserID, err := uuid.Parse(otherUserIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid other_user_id"})
			return
		}

		var referralID *uuid.UUID
		if refIDStr := c.Query("referral_id"); refIDStr != "" {
			parsed, err := uuid.Parse(refIDStr)
			if err == nil {
				referralID = &parsed
			}
		}

		convID, err = h.chatUC.GetConversationID(c.Request.Context(), userID, otherUserID, referralID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	messages, total, err := h.chatUC.GetMessages(c.Request.Context(), userID, convID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedChatResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Message history retrieved successfully",
		},
		Data:     dto.ToChatMessageResponseSlice(messages),
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}

type MarkReadRequest struct {
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       *uuid.UUID `json:"sender_id"` // Deprecated but supported for backward compatibility
}

// MarkRead godoc
// @Summary      Mark Messages as Read
// @Description  Mark all incoming messages in a specific conversation as read for the authenticated user.
// @Description  
// @Description  ### Behavior:
// @Description  - Proceeds successfully even if the conversation is referral-scoped terminal (read-only) or administratively locked (disabled) so that users can clear their badges.
// @Description  - Supports backward compatibility by accepting either `conversation_id` or `sender_id` (deprecated).
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        request  body  MarkReadRequest  true  "Mark read options containing conversation_id or sender_id"
// @Success      200  {object}  dto.BaseResponse
// @Failure      400  {object}  dto.ErrorResponse  "Invalid payload, missing both IDs, or malformed UUIDs"
// @Failure      500  {object}  dto.ErrorResponse  "Internal update operation error"
// @Security     BearerAuth
// @Router       /api/v1/chat/messages/read [post]
func (h *ChatHandler) MarkRead(c *gin.Context) {
	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	var convID uuid.UUID
	var err error

	if req.ConversationID != uuid.Nil {
		convID = req.ConversationID
	} else if req.SenderID != nil {
		convID, err = h.chatUC.GetConversationID(c.Request.Context(), userID, *req.SenderID, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "either conversation_id or sender_id is required"})
		return
	}

	if err := h.chatUC.MarkRead(c.Request.Context(), userID, convID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Messages marked as read"})
}

// GetUnreadCount godoc
// @Summary      Get Total Unread Message Count
// @Description  Retrieve the total sum of all unread chat messages targeting the authenticated user across all conversations.
// @Description  
// @Description  ### Roles:
// @Description  - Accessible by all authenticated roles except `MOH_ANALYST`.
// @Tags         Chat
// @Produce      json
// @Success      200  {object}  dto.ChatUnreadCountResponse
// @Failure      401  {object}  dto.ErrorResponse  "Unauthorized user session"
// @Failure      500  {object}  dto.ErrorResponse  "Unread count calculation failure"
// @Security     BearerAuth
// @Router       /api/v1/chat/unread-count [get]
func (h *ChatHandler) GetUnreadCount(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	count, err := h.chatUC.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Unread count retrieved successfully",
		"unread_count": count,
	})
}

type ToggleDisabledRequest struct {
	IsDisabled bool   `json:"is_disabled"`
	Reason     string `json:"reason" binding:"required"`
}

// ToggleDisabled godoc
// @Summary      Toggle Conversation Lock State (Admin)
// @Description  Administratively lock or unlock a chat channel. Locking a channel (`is_disabled = true`) prevents any further message sending while maintaining read history access.
// @Description  
// @Description  ### Role & Boundary Security:
// @Description  - **System Super Admin:** Authorized to lock/unlock any conversation globally without restriction.
// @Description  - **Hospital Admin:** Authorized to lock/unlock conversations **only** if they involve a participant or a referral associated with their hospital. Access from outside this boundary is rejected.
// @Description  - **Other Roles:** Rejected with `403 Forbidden`.
// @Description  - **Audit Logging:** Triggers an immutable system audit log entry capturing the change state and reason.
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        id       path      string                 true  "Conversation UUID to modify"
// @Param        request  body      ToggleDisabledRequest  true  "Lock state toggle (`is_disabled`) and mandatory action justification reason"
// @Success      200      {object}  dto.BaseResponse
// @Failure      400      {object}  dto.ErrorResponse  "Invalid conversation ID, or missing required justification reason"
// @Failure      403      {object}  dto.ErrorResponse  "Unauthorized admin boundary violation, or non-admin role attempt"
// @Failure      500      {object}  dto.ErrorResponse  "Internal update or audit logging failure"
// @Security     BearerAuth
// @Router       /api/v1/chat/conversations/{id}/toggle-disabled [put]
func (h *ChatHandler) ToggleDisabled(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid conversation ID"})
		return
	}

	var req ToggleDisabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	userIdVal, _ := c.Get("userID")
	adminID, _ := userIdVal.(uuid.UUID)

	if err := h.chatUC.ToggleDisabled(c.Request.Context(), adminID, convID, req.IsDisabled, req.Reason); err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Conversation toggle updated successfully"})
}

// DeleteConversation godoc
// @Summary      Soft-delete Conversation (Admin)
// @Description  Administratively soft-delete a chat conversation, hiding it from participants while preserving GORM DB records for clinical auditing.
// @Description  
// @Description  ### Role & Boundary Security:
// @Description  - **System Super Admin:** Global permission to soft-delete any conversation.
// @Description  - **Hospital Admin:** Authorized **only** for conversations involving their hospital's staff or active referrals.
// @Description  - **Other Roles:** Rejected with `403 Forbidden`.
// @Description  - **Audit Trail:** Registers a system audit log capturing the deletion event.
// @Tags         Chat
// @Produce      json
// @Param        id   path      string  true  "Conversation UUID to soft-delete"
// @Success      200  {object}  dto.BaseResponse
// @Failure      400  {object}  dto.ErrorResponse  "Invalid or malformed conversation ID"
// @Failure      403  {object}  dto.ErrorResponse  "Unauthorized boundary violation, or non-admin role attempt"
// @Failure      500  {object}  dto.ErrorResponse  "Internal soft-delete execution failure"
// @Security     BearerAuth
// @Router       /api/v1/chat/conversations/{id} [delete]
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid conversation ID"})
		return
	}

	userIdVal, _ := c.Get("userID")
	adminID, _ := userIdVal.(uuid.UUID)

	if err := h.chatUC.SoftDeleteConversation(c.Request.Context(), adminID, convID); err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BaseResponse{Success: true, Message: "Conversation soft-deleted successfully"})
}

// ListContacts godoc
// @Summary      List Eligible Chat Contacts
// @Description  Retrieve a paginated and searchable list of active users that the authenticated caller is legally authorized to chat with based on role visual boundaries, active access grants, and clinical referral relationships.
// @Description  
// @Description  ### Contact Search Modes:
// @Description  
// @Description  #### 1. Same-Hospital Mode (Without `referral_id`)
// @Description  Returns colleagues working at the **same hospital** as the caller:
// @Description  - **Self-Exclusion:** The caller is never included in the contact list.
// @Description  - **Specialist Visibility:** Specialists can only see referring doctors to whom they have **active access** (via accepted referrals or explicit access grants).
// @Description  - **Liaison Officer / Receptionist / Dept Head:** Follow standard hospital-wide visual scopes.
// @Description  - **Admin & Super-Admin:** Standard administrative and super-admin visual bounds.
// @Description  - **System Constraints:** Excludes `MOH_ANALYST` completely and excludes `SYSTEM_SUPER_ADMIN` (unless the caller themselves is a super-admin).
// @Description  
// @Description  #### 2. Cross-Hospital Mode (With `referral_id`)
// @Description  Returns eligible clinical contacts at the **opposite hospital** involved in that specific referral context:
// @Description  - **Visually Scoped Roles:** Limit eligible chat contacts to:
// @Description    - The referring doctor who created/managed the referral.
// @Description    - Liaison officers at the target hospital.
// @Description    - Specialists with active access or assignment to the referral.
// @Description    - The target department head and the target hospital administrator.
// @Description  - **Access Gate:** The caller must have permission to access the referral to use this mode.
// @Description  
// @Description  ### Filtering & Search Options:
// @Description  - **`search`:** Filter users dynamically by first name, middle name, or last name.
// @Description  - **`role`:** Limit contacts to a specific user role (e.g., `RECEIVING_SPECIALIST`, `REFERRING_DOCTOR`).
// @Description  
// @Description  ### System Constraints:
// @Description  - `MOH_ANALYST` users are completely barred from using or being returned by this endpoint.
// @Tags         Chat
// @Produce      json
// @Param        referral_id query string false "Referral UUID to list eligible cross-hospital contacts for a referral"
// @Param        search      query string false "Search pattern targeting first, middle, or last names of contacts"
// @Param        role        query string false "Filter by specific user role (e.g., REFERRING_DOCTOR)"
// @Param        limit       query int    false "Pagination limit (default 50)"
// @Param        page        query int    false "Page number (default 1)"
// @Success      200 {object} dto.PaginatedContactResponse
// @Failure      400 {object} dto.ErrorResponse "Invalid referral ID, unauthorized role, or access boundaries violation"
// @Failure      403 {object} dto.ErrorResponse "MOH Analyst role completely blocked"
// @Failure      500 {object} dto.ErrorResponse "Internal database lookup error"
// @Security     BearerAuth
// @Router       /api/v1/chat/contacts [get]
func (h *ChatHandler) ListContacts(c *gin.Context) {
	userIdVal, _ := c.Get("userID")
	userID, _ := userIdVal.(uuid.UUID)

	userRoleVal, _ := c.Get("role")
	roleStr, _ := userRoleVal.(string)
	role := entity.UserRole(roleStr)

	hospIdVal, _ := c.Get("hospID")
	hospID := uuid.Nil
	if hID, ok := hospIdVal.(uuid.UUID); ok {
		hospID = hID
	} else if hID, ok := hospIdVal.(*uuid.UUID); ok && hID != nil {
		hospID = *hID
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var referralID *uuid.UUID
	if refIDStr := c.Query("referral_id"); refIDStr != "" {
		parsed, err := uuid.Parse(refIDStr)
		if err == nil {
			referralID = &parsed
		} else {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: "invalid referral_id format"})
			return
		}
	}

	search := c.Query("search")

	var roleFilter *entity.UserRole
	if rFilterStr := c.Query("role"); rFilterStr != "" {
		parsed := entity.UserRole(rFilterStr)
		roleFilter = &parsed
	}

	contacts, total, err := h.chatUC.ListContacts(c.Request.Context(), userID, role, hospID, referralID, search, roleFilter, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.PaginatedContactResponse{
		BaseResponse: dto.BaseResponse{
			Success: true,
			Message: "Eligible contacts retrieved successfully",
		},
		Data:     contacts,
		Total:    total,
		Page:     page,
		PageSize: limit,
	})
}
