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
		err := fmt.Errorf("pricing id is required")
		message := echo.Map{
			"message": "failed to get pricing",
			"error":   err.Error(),
		}
		s.logger.Error(err)

		return c.JSON(http.StatusBadRequest, message)
	}

	pricing, err := s.db.FindPricingById(c.Request().Context(), pricingID)
	if err != nil {
		message := echo.Map{
			"message": "failed to get pricing",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, pricing)
}

func (s *Server) CreatePricing(c echo.Context) error {
	var pricing types.PricingCreateDto
	if err := c.Bind(&pricing); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := c.Validate(&pricing); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": err.Error(),
		})
	}

	created, err := s.db.CreatePricing(c.Request().Context(), pricing)
	if err != nil {
		message := echo.Map{
			"message": "failed to create pricing",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) DeletePricing(c echo.Context) error {
	pricingID := c.Param("pricingId")
	if pricingID == "" {
		message := echo.Map{
			"message": "failed to delete pricing",
			"error":   "pricing id is required",
		}
		return c.JSON(http.StatusBadRequest, message)
	}

	err := s.db.DeletePricingById(c.Request().Context(), pricingID)
	if err != nil {
		message := echo.Map{
			"message": "failed to delete pricing",
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.NoContent(http.StatusNoContent)
}
