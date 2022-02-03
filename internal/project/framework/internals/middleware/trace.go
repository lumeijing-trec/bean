/**#bean*/ /*#bean.replace({{ .Copyright }})**/
package middleware

import (
	"fmt"

	/**#bean*/
	"demo/framework/internals/helpers"
	/*#bean.replace("{{ .PkgPath }}/framework/internals/helpers")**/

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// Tracer attach a root sentry span context to the request.
func Tracer() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Start a sentry span for tracing.
			span := sentry.StartSpan(c.Request().Context(), "REST API",
				sentry.TransactionName(fmt.Sprintf(c.Request().RequestURI)),
				sentry.ContinueFromRequest(c.Request()),
			)
			span.Description = helpers.CurrFuncName()
			defer span.Finish()
			r := c.Request().Clone(span.Context())
			c.SetRequest(r)
			return next(c)
		}
	}
}
