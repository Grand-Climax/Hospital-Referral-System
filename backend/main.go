package main

import (
	"Hospital-Referral-System/config"
	"Hospital-Referral-System/internal/delivery/http/handlers"
	router "Hospital-Referral-System/internal/delivery/http/routes"
	"Hospital-Referral-System/internal/infrastructure/database"
	"Hospital-Referral-System/internal/repository"
	"Hospital-Referral-System/internal/usecase"
)

func main() {
	cfg := config.LoadConfig()
	database.ConnectDB(cfg)

	//setup the role api
	rolerepo := repository.NewRoleRepository(database.DB)
	roleusecase := usecase.NewRoleUsecase(rolerepo)
	rolehandler := handlers.NewRoleHandler(roleusecase)
	

	r := router.SetupRouter(&router.RouterConfig{RoleHandler: rolehandler})

	r.Run()
}