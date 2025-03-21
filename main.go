package main

// D.L. Bowen, L.M. Byrd, W.F. Clocksin - A Portable Prolog Compiler

import (
    "fmt"
)

func main() {
    s := MustParseRules(`
    append(nil, L, L).
    append(cons(X,L1), L2, cons(X,L3)) :- append(L1, L2, L3).`)
    procs := compileProcedures(s)
    i := NewInterpreter(procs)

    q := `append(cons(a, cons(b, nil)), cons(c, nil), L)`
    b := i.interpret(q)  // expect: L = cons(a, cons(b, cons(c, nil))).
    for _, ans := range b {
        fmt.Println("L =", ans["L"])
    }
    fmt.Println("fail")
    fmt.Println()

    q = `append(L, X, cons(a, cons(b, cons(c, nil))))`
    b = i.interpret(q)
    for _, ans := range b {
        fmt.Println("L =", ans["L"])
        fmt.Println("X =", ans["X"])
    }
    fmt.Println("fail")
}
