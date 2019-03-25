package master

import (
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/edgestore/edgestore/association"

	"github.com/edgestore/edgestore/entity"
	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/edgestore/edgestore/internal/server"
	"github.com/gin-gonic/gin"
)

const Prefix = "/api/v1"

const DefaultPaginationLimit = 10

func NewPagination(ctx *gin.Context) *model.Pagination {
	perPage, _ := strconv.Atoi(ctx.DefaultQuery("per_page", strconv.Itoa(DefaultPaginationLimit)))
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "0"))
	return model.NewPagination(perPage, page)
}

func (s *service) AbortWithError(ctx *gin.Context, err error) {
	s.logger.Error(err)
	res := ER(err)
	ctx.AbortWithStatusJSON(res.Code, res)
}

func (s *service) HTTPHandler() http.Handler {
	handler := gin.New()
	handler.Use(gin.Recovery())

	handler.Use(server.CORSHandler())
	handler.Use(server.LoggerHandler(s.logger, time.RFC3339, true))
	handler.Use(server.RequestIDHandler())
	handler.NoRoute(server.NotFoundHandler)
	handler.GET("/", s.RootHandler)

	api := handler.Group(Prefix).Use(NewTenantMiddleware())
	api.DELETE("/associations/:id", s.DeleteAssociationHandler)
	api.GET("/associations/:id", s.GetAssociationHandler)
	api.POST("/associations", s.CreateAssociationHandler)
	api.PUT("/associations/:id", s.UpdateAssociationHandler)

	api.DELETE("/entities/:id", s.DeleteEntityHandler)
	api.GET("/entities/:id", s.GetEntityHandler)
	api.POST("/entities", s.CreateEntityHandler)
	api.PUT("/entities/:id", s.UpdateEntityHandler)

	api.POST("/guid", s.CreateGUIDHandler)

	return handler
}

func (s *service) GetAssociationHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.GetAssociationHandler"

	tenant := ctx.GetString(TenantKey)
	id := ctx.Param("id")

	if agg, err := s.association.GetAssociation(ctx, model.ID(id), model.ID(tenant)); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		ctx.JSON(http.StatusOK, agg)
	}
}

func (s *service) CreateAssociationHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.CreateAssociationHandler"

	var form association.InsertAssociation
	if err := ctx.ShouldBind(&form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, errors.E(op, errors.Invalid, err))
		return
	}

	tenant := ctx.GetString(TenantKey)
	form.TenantID = model.ID(tenant)

	if err := s.association.CreateAssociation(ctx, &form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		location := path.Join(Prefix, "entities", string(form.ID))
		ctx.Header("Location", location)
		ctx.Writer.WriteHeader(http.StatusAccepted)
	}
}

func (s *service) UpdateAssociationHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.UpdateAssociationHandler"

	var form association.UpdateAssociation
	if err := ctx.ShouldBind(&form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, errors.E(op, errors.Invalid, err))
		return
	}

	tenant := ctx.GetString(TenantKey)
	form.TenantID = model.ID(tenant)
	form.ID = model.ID(ctx.Param("id"))
	if err := s.association.UpdateAssociation(ctx, &form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		ctx.Writer.WriteHeader(http.StatusAccepted)
	}
}

func (s *service) DeleteAssociationHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.DeleteAssociationHandler"

	form := association.DeleteAssociation{}
	tenant := ctx.GetString(TenantKey)
	form.TenantID = model.ID(tenant)
	form.ID = model.ID(ctx.Param("id"))

	if err := s.association.DeleteAssociation(ctx, &form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		ctx.Writer.WriteHeader(http.StatusAccepted)
	}
}

func (s *service) GetEntityHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.GetEntityHandler"

	tenant := ctx.GetString(TenantKey)
	id := ctx.Param("id")

	_, dataOnly := ctx.GetQuery("data")

	if agg, err := s.entity.GetEntity(ctx, model.ID(id), model.ID(tenant)); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		if dataOnly {
			ctx.JSON(http.StatusOK, agg.Data)
		} else {
			ctx.JSON(http.StatusOK, agg)
		}

	}
}

func (s *service) CreateEntityHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.CreateEntityHandler"

	var form entity.InsertEntity
	if err := ctx.ShouldBind(&form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, errors.E(op, errors.Invalid, err))
		return
	}

	tenant := ctx.GetString(TenantKey)
	form.TenantID = model.ID(tenant)

	if err := s.entity.CreateEntity(ctx, &form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		location := path.Join(Prefix, "entities", string(form.ID))
		ctx.Header("Location", location)
		ctx.Writer.WriteHeader(http.StatusAccepted)
	}
}

func (s *service) UpdateEntityHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.UpdateEntityHandler"

	var form entity.UpdateEntity
	if err := ctx.ShouldBind(&form); err != nil {
		s.AbortWithError(ctx, errors.E(op, errors.Invalid, err))
		return
	}

	tenant := ctx.GetString(TenantKey)
	form.TenantID = model.ID(tenant)
	form.ID = model.ID(ctx.Param("id"))
	if err := s.entity.UpdateEntity(ctx, &form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		ctx.Writer.WriteHeader(http.StatusAccepted)
	}
}

func (s *service) DeleteEntityHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.DeleteEntityHandler"

	form := entity.DeleteEntity{}
	tenant := ctx.GetString(TenantKey)
	form.TenantID = model.ID(tenant)
	form.ID = model.ID(ctx.Param("id"))

	if err := s.entity.DeleteEntity(ctx, &form); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		ctx.Writer.WriteHeader(http.StatusAccepted)
	}
}

func (s *service) CreateGUIDHandler(ctx *gin.Context) {
	const op errors.Op = "api/service.CreateGUIDHandler"

	if id, err := s.guid.NextID(); err != nil {
		s.logger.Error(errors.E(op, err))
		s.AbortWithError(ctx, err)
	} else {
		location := path.Join(Prefix, "guid", id)
		ctx.Header("Location", location)
		ctx.JSON(http.StatusOK, gin.H{
			"id":         id,
			"machine":    s.cfg.MachineID,
			"created_at": time.Now(),
		})
	}
}
