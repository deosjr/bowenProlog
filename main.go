package main

// D.L. Bowen, L.M. Byrd, W.F. Clocksin - A Portable Prolog Compiler

import (
    "fmt"
)

func main() {
    s := MustParseRules(`
    append(nil, L, L).
    append(cons(X,L1), L2, cons(X,L3)) :- append(L1, L2, L3).`)

    for _, r := range s {
        fmt.Printf("%s\n", r)
    }

    procs := compileProcedures(s)
    for _, p := range procs {
        fmt.Printf("%v\n", p)
    }

    i := NewInterpreter(procs)
    q := `append(cons(a, cons(b, nil)), cons(c, nil), L)`
    b := i.interpret(q)  // expect: L = cons(a, cons(b, cons(c, nil))).
    for _, ans := range b {
        fmt.Println("L =", ans["L"])
    }
    fmt.Println("fail")
}
