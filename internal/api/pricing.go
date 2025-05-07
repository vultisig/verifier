package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/types"
)

func (s *Server) GetPricing(c echo.Context) error {
	pricingID := c.Param("pricingId")
	if pricingID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid pricing id"))
	}

	pricing, err := s.db.FindPricingById(c.Request().Context(), pricingID)
	if err != nil {
		s.logger.Errorf("error finding pricing %s: %v", pricingID, err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get pricing"))
	}

	return c.JSON(http.StatusOK, pricing)
}

func (s *Server) CreatePricing(c echo.Context) error {
	var pricing types.PricingCreateDto
	if err := c.Bind(&pricing); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
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

	err := s.db.DeletePricingById(c.Request().Context(), pricingID)
	if err != nil {
		s.logger.Errorf("error deleting pricing %s: %s", pricingID, err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to delete pricing"))
	}

	return c.NoContent(http.StatusNoContent)
}
