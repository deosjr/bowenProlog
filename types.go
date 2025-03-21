package main

import (
    "fmt"
    "strings"
)

type expression interface {
    PrintExpression() string
}

type variable int64

func (v variable) PrintExpression() string {
    return fmt.Sprintf("v#%d", v)
}

type number int64

func (n number) PrintExpression() string {
    return fmt.Sprintf("%d", n)
}

// TODO: investigate stdlib unique to intern strings
type symbol string

func (s symbol) PrintExpression() string {
    return string(s)
}

const (
    emptylist   symbol = "nil"
    underscore  symbol = "_"
    true_value  symbol = "true"
    false_value symbol = "false"
)

type list struct {
    head expression // can be anything
    tail expression // has to be list or emptylist!
}

func (l list) PrintExpression() string {
    if l.tail == emptylist {
        return fmt.Sprintf("[%s]", l.head.PrintExpression())
    }
    return fmt.Sprintf("[%s|%s]", l.head.PrintExpression(), l.tail.PrintExpression())
}

type process struct {
    functor string
    args []expression
}

func (p process) arity() int {
    return len(p.args)
}

func (p process) isPredefined() bool {
    // only predefined functors for now
    return p.functor == ":=" || p.functor == "isplus" || p.functor == "is"
}

func (p process) isInfix() bool {
    return p.functor == ":=" || p.functor == "is"
}

func (p process) String() string {
    return p.PrintExpression()
}

func (p process) PrintExpression() string {
    args := []string{}
    for _, arg := range p.args {
        args = append(args, arg.PrintExpression())
    }
    if p.isInfix() && len(args) == 2 {
        return fmt.Sprintf("%s %s %s", args[0], p.functor, args[1])
    }
    return fmt.Sprintf("%s(%s)", p.functor, strings.Join(args, ","))
}

type rule struct {
    head process
    body []process
}

func (r rule) String() string {
    if len(r.body) == 0 {
        return fmt.Sprintf("%s.", r.head)
    }
    body := []string{}
    for _, p := range r.body {
        body = append(body, p.String())
    }
    return fmt.Sprintf("%s :- %s.", r.head, strings.Join(body, ","))
}
