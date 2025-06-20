package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) GetPricing(c echo.Context) error {
	pricingID := c.Param("pricingId")
	if pricingID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "invalid pricing id", "pricing id cannot be null"))
	}
	uPricingID, err := uuid.Parse(pricingID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "invalid pricing id", err.Error()))
	}
	pricing, err := s.db.FindPricingById(c.Request().Context(), uPricingID)
	if err != nil {
		s.logger.Errorf("error finding pricing %s: %v", pricingID, err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get pricing", err.Error()))
	}

	return c.JSON(http.StatusOK, pricing)
}
