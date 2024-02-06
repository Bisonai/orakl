package reporter

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	reporter := router.Group("/reporter")

	reporter.Post("", insert)
	reporter.Get("", get)
	reporter.Get("/oracle-address/:oracleAddress", getByOracleAddress)
	reporter.Get("/:id", getById)
	reporter.Patch("/:id", updateById)
	reporter.Delete("/:id", deleteById)
}
