package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
)

type (
	CommonHandler struct {
		Path         string
		HandlerWorks map[string]func(*gin.Context)
	}

	HandlerCallBack func(path string, methodSet []string, fun func(*gin.Context))

	Handler interface {
		AccessPath() string
		ForeachHandler(callback func(path string, methodSet []string, fun func(*gin.Context)))
	}
)

func (p *CommonHandler) AccessPath() string {
	return p.Path
}

func (p *CommonHandler) ForeachHandler(callback func(path string, methodSet []string, fun func(*gin.Context))) {
	for uFlag, handler := range p.HandlerWorks {
		path, method := p.ParserHandlerMapKey(uFlag)
		callback(path, []string{method}, handler)
	}
}

func (p *CommonHandler) ParserHandlerMapKey(uFlag string) (path, method string) {
	fs := strings.Split(uFlag, "\001")
	path = fs[0]
	method = fs[1]
	return
}

func (p *CommonHandler) requestMapping(path, method string, handlerWork func(*gin.Context)) {
	if p.HandlerWorks == nil {
		p.HandlerWorks = make(map[string]func(*gin.Context))
	}
	uFlag := fmt.Sprintf("%s\001%s", path, method)
	p.HandlerWorks[uFlag] = handlerWork
}

func (p *CommonHandler) getMapping(path string, handlerWork func(*gin.Context)) {
	p.requestMapping(path, "GET", handlerWork)
}

func (p *CommonHandler) postMapping(path string, handlerWork func(*gin.Context)) {
	p.requestMapping(path, "POST", handlerWork)
}

func (p *CommonHandler) putMapping(path string, handlerWork func(*gin.Context)) {
	p.requestMapping(path, "PUT", handlerWork)
}

func (p *CommonHandler) deleteMapping(path string, handlerWork func(*gin.Context)) {
	p.requestMapping(path, "DELETE", handlerWork)
}
