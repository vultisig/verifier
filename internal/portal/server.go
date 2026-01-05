package portal

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage/postgres/queries"
)

type Server struct {
	cfg     config.PortalConfig
	queries *queries.Queries
	logger  *logrus.Logger
}

func NewServer(cfg config.PortalConfig, pool *pgxpool.Pool) *Server {
	return &Server{
		cfg:     cfg,
		queries: queries.New(pool),
		logger:  logrus.WithField("service", "portal").Logger,
	}
}

func (s *Server) Start() error {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	s.registerRoutes(e)

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.logger.Infof("Starting portal server on %s", addr)
	return e.Start(addr)
}

func (s *Server) registerRoutes(e *echo.Echo) {
	e.GET("/plugins/:id", s.GetPlugin)
}

func (s *Server) GetPlugin(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	plugin, err := s.queries.GetPluginByID(c.Request().Context(), queries.PluginID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to get plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, plugin)
}
