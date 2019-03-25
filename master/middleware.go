package master

import (
	"net/http"

	"github.com/edgestore/edgestore/internal/server"
	"github.com/gin-gonic/gin"
)

const TenantKey = "tenant"

func NewTenantMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tenant := ctx.GetHeader("Edgestore-Tenant")

		if tenant != "" {
			ctx.Set(TenantKey, tenant)
			ctx.Next()
			return
		}

		server.Abort(ctx, http.StatusUnauthorized, "Invalid Tenant ID. Make sure to provide a valid X-Edgestore-Tenant header.")
	}
}
