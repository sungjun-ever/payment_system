package product

import (
	"payment_system/internal/pkg/apperr"
	"payment_system/internal/pkg/response"
	productDomain "payment_system/internal/product/domain"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	ps *ProductService
}

func NewProductHandler(ps *ProductService) *ProductHandler {
	return &ProductHandler{ps}
}

func (p *ProductHandler) Create(c *gin.Context) {
	var dto productDomain.CreatRequest
	if err := c.ShouldBindJSON(&dto); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	created, err := p.ps.CreateProduct(c.Request.Context(), dto)

	if err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 500, apperr.S001, err, nil))
		return
	}

	response.ToSuccessResponse(c, 201, created)
}

func (p *ProductHandler) Get(c *gin.Context) {
	var dto productDomain.UriRequest

	if err := c.ShouldBindUri(&dto); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	pd, err := p.ps.GetProduct(c.Request.Context(), dto)

	if err != nil {
		_ = c.Error(apperr.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 200, pd)
}

func (p *ProductHandler) Update(c *gin.Context) {
	var uri productDomain.UriRequest
	var dto productDomain.UpdateRequest

	if err := c.ShouldBindUri(&uri); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	if err := c.ShouldBindJSON(&dto); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	dto.ID = uri.ID

	pd, err := p.ps.UpdateProduct(c.Request.Context(), dto)

	if err != nil {
		_ = c.Error(apperr.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 200, pd)
}

func (p *ProductHandler) Delete(c *gin.Context) {
	var uri productDomain.UriRequest

	if err := c.ShouldBindUri(&uri); err != nil {
		_ = c.Error(apperr.NewAppError(apperr.LevelError, 400, apperr.C001, err, nil))
		return
	}

	err := p.ps.DeleteProduct(c.Request.Context(), uri)

	if err != nil {
		_ = c.Error(apperr.ToAppError(err))
		return
	}

	response.ToSuccessResponse(c, 200, nil)
}
