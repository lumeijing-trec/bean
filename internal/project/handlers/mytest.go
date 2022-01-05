/**#bean*/ /*#bean.replace({{ .Copyright }})**/
package handlers

import (
	"net/http"

	/**#bean*/
	"demo/services"
	/*#bean.replace("{{ .PkgPath }}/services")**/

	"github.com/labstack/echo/v4"
)

type MyTestHandler interface {
	MyTestIndex(c echo.Context) error
}

type myTestHandler struct {
	myTestService services.MyTestService
}

func NewMyTestHandler(myTestSvc services.MyTestService) *myTestHandler {
	return &myTestHandler{myTestSvc}
}

func (handler *myTestHandler) MyTestIndex(c echo.Context) error {

	res, _ := handler.myTestService.GetMasterSQLTableList(c)

	c.Logger().Info(res["dbName"])

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Howdy! I am {{ .PkgName }} 🚀 ",
	})
}
