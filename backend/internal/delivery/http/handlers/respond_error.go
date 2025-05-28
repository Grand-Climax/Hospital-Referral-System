package handlers

import "github.com/gin-gonic/gin"

func RespondErrors(ctx *gin.Context, code int, message string, err error){
	ctx.IndentedJSON(code, gin.H{"message": message, "error": err.Error()})
}

func Respond(ctx *gin.Context, code int, message string){
	ctx.IndentedJSON(code, gin.H{"message": message})
}