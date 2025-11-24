package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) GetPricing(c echo.Context) error {
	pricingID := c.Param("pricingId")
	if pricingID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPricingID))
	}
	uPricingID, err := uuid.Parse(pricingID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPricingID))
	}
	pricing, err := s.db.FindPricingById(c.Request().Context(), uPricingID)
	if err != nil {
		s.logger.WithError(err).Errorf("error finding pricing %s", pricingID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPricingGetFailed))
	}
	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, pricing))
}
