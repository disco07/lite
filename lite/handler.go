package lite

import (
	"context"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"log/slog"
	"net/http"
	"regexp"
	"swagger/codec"
)

func fiberHandler[ResponseBody, RequestBody any, E codec.Encoder[ResponseBody]](
	controller func(*ContextWithBody[RequestBody]) (ResponseBody, error),
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		contextWithBody := &ContextWithBody[RequestBody]{Ctx: c}
		response, err := controller(contextWithBody)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		var encoder E

		return encoder.Encode(c, response)
	}
}

func Get[ResponseBody, RequestBody any, E codec.Encoder[ResponseBody]](
	app *App,
	path string,
	controller func(*ContextWithBody[RequestBody]) (ResponseBody, error),
	middleware ...fiber.Handler,
) Route[ResponseBody, RequestBody] {
	var encoder E

	return registerRoute[ResponseBody, RequestBody](
		app,
		Route[ResponseBody, RequestBody]{path: path, method: http.MethodGet, contentType: encoder.ContentType()},
		fiberHandler[ResponseBody, RequestBody, E](controller),
		middleware...,
	)
}

func registerRoute[ResponseBody, RequestBody any](
	app *App,
	route Route[ResponseBody, RequestBody],
	controller fiber.Handler,
	middleware ...fiber.Handler,
) Route[ResponseBody, RequestBody] {
	if len(middleware) > 0 {
		app.Add(route.method,
			route.path,
			middleware...,
		)
	}

	app.Add(
		route.method,
		route.path,
		otelfiber.Middleware(
		//otelfiber.WithPort(route.cfg.Port()),
		//otelfiber.WithTracerProvider(route.tracerProvider),
		),
		controller,
	)

	status := getStatusCode(route.method)

	operation, err := RegisterOpenAPIOperation[ResponseBody, RequestBody](app, route.method, route.path, route.contentType, status)
	if err != nil {
		slog.ErrorContext(context.Background(), "failed to register openapi operation", slog.Any("error", err))
		panic(err)
	}

	route.Operation = operation

	return route
}

// parseRoutePath parses the route path and returns the path and the query parameters.
// Example : /item/:user/:id -> /item/{user}/{id}
func parseRoutePath(route string) (string, []string) {
	// Define a regular expression to match route parameters
	re := regexp.MustCompile(`:([a-zA-Z0-9_]+)`)

	// Find all matches of the parameters in the route
	matches := re.FindAllStringSubmatch(route, -1)

	var params []string
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	// Replace all instances of :param with {param}
	modifiedRoute := re.ReplaceAllString(route, `{$1}`)

	return modifiedRoute, params
}

// Get status code from the method
func getStatusCode(method string) int {
	switch method {
	case http.MethodGet:
		return http.StatusOK
	case http.MethodPost:
		return http.StatusCreated
	case http.MethodPut:
		return http.StatusOK
	case http.MethodDelete:
		return http.StatusNoContent
	case http.MethodPatch:
		return http.StatusOK
	default:
		return http.StatusOK
	}
}
