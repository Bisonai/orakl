package sign

import (
	"errors"
	"time"

	"bisonai.com/orakl/node/pkg/delegator/metrics"
	"github.com/gofiber/fiber/v2"
)

const signResultKey = "signResult"

// notAllowlistedError is a contract or function missing from the allowlist,
// as opposed to a failed allowlist lookup.
type notAllowlistedError string

func (e notAllowlistedError) Error() string { return string(e) }

// observeSign records the result and latency of a POST /sign/v2 request. It
// runs before RequireToken so auth rejections are counted too.
func observeSign(c *fiber.Ctx) (err error) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			// re-panicked for the app's recover middleware
			metrics.ObserveSign(metrics.ResultError, tokenName(c), time.Since(start))
			panic(r)
		}
		metrics.ObserveSign(signResult(c, err), tokenName(c), time.Since(start))
	}()
	return c.Next()
}

// reject tags err with the sign result it maps to.
func reject(c *fiber.Ctx, result string, err error) error {
	c.Locals(signResultKey, result)
	return err
}

// rejectInvalid tags err as invalid_tx and answers 400: the client sent it.
func rejectInvalid(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if !errors.As(err, &fe) {
		err = fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return reject(c, metrics.ResultInvalidTx, err)
}

// signResult maps the handler outcome to a result label. Untagged errors are
// "error".
func signResult(c *fiber.Ctx, err error) string {
	if result, ok := c.Locals(signResultKey).(string); ok {
		return result
	}
	if err == nil && c.Response().StatusCode() == fiber.StatusOK {
		return metrics.ResultOK
	}
	return metrics.ResultError
}

// storeError counts a failed store operation and tags the request as "error".
func storeError(c *fiber.Ctx, op string, err error) error {
	metrics.StoreError(op)
	return reject(c, metrics.ResultError, err)
}
