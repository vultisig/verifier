package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/types"
)

func (s *Server) GetPricing(c echo.Context) error {
	pricingID := c.Param("pricingId")
	if pricingID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid pricing id"))
	}
	uPricingID, err := uuid.Parse(pricingID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid pricing id"))
	}
	pricing, err := s.db.FindPricingById(c.Request().Context(), uPricingID)
	if err != nil {
		s.logger.Errorf("error finding pricing %s: %v", pricingID, err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get pricing"))
	}

	return c.JSON(http.StatusOK, pricing)
}

func (s *Server) CreatePricing(c echo.Context) error {
	var pricing types.PricingCreateDto
	if err := c.Bind(&pricing); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to parse request"))
	}

	if err := c.Validate(&pricing); err != nil {
		s.logger.Errorf("failed to validate pricing: %v", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid pricing data"))
	}

	created, err := s.db.CreatePricing(c.Request().Context(), pricing)
	if err != nil {
		s.logger.Errorf("failed to create pricing: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create pricing"))
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) DeletePricing(c echo.Context) error {
	pricingID := c.Param("pricingId")
	if pricingID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid pricing id"))
	}
	uPricingID, err := uuid.Parse(pricingID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid pricing id"))
	}
	if err := s.db.DeletePricingById(c.Request().Context(), uPricingID); err != nil {
		s.logger.Errorf("error deleting pricing %s: %s", pricingID, err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to delete pricing"))
	}

	return c.NoContent(http.StatusNoContent)
}
