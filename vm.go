package main

import (
    "fmt"
)

// partial overlap with parsing types, but let's keep them separate

type procedure struct {
    name    string
    arity   int
    clauses []clause
}

func (p procedure) String() string {
    return fmt.Sprintf("procedure(%s/%d, %v)", p.name, p.arity, p.clauses)
}

type clause struct {
    xrTable xrTable
    numVars int
    bytecodes []instruction
}

type instruction int64

const (
    CONST instruction = iota
    VAR
    FUNCTOR
    POP
    ENTER
    CALL
    EXIT
)

type xrTable []entry

type entry interface {
    printEntry() string
}

type integer number

func (i integer) printEntry() string {
    return fmt.Sprintf("%d", i)
}

type atom symbol

func (a atom) printEntry() string {
    return fmt.Sprintf("%s", a)
}

func constant(v any) entry {
    switch t := v.(type) {
    case int64:
        return integer(t)
    case string:
        return atom(t)
    }
    panic(fmt.Sprintf("unexepected constant %v", v))
}

// a compound term
type functorEntry struct {
    name    string
    arity   int
}

func (f functorEntry) printEntry() string {
    return fmt.Sprintf("%s/%d", f.name, f.arity)
}

func functor(name string, arity int) functorEntry {
    return functorEntry{name, arity}
}

// a procedure call
type procEntry struct {
    functorEntry
}

func proc(name string, arity int) procEntry {
    return procEntry{functorEntry{name, arity}}
}

type procKey struct {
    name    string
    arity   int
}

func compileProcedures(rules []rule) []procedure {
    m := map[procKey][]rule{}
    for _, r := range rules {
        k := procKey{r.head.functor, r.head.arity()}
        m[k] = append(m[k], r)
    }
    procedures := []procedure{}
    for _, rules := range m {
        p := compileProcedure(rules)
        procedures = append(procedures, p)
    }
    return procedures
}

func compileProcedure(rules []rule) procedure {
    clauses := []clause{}
    for _, r := range rules {
        clauses = append(clauses, compileClause(r))
    }
    head := rules[0].head
    return procedure{name: head.functor, arity: head.arity(), clauses: clauses}
}

func compileClause(r rule) clause {
    xrMap := map[entry]int{}
    byteCodes := compileArgs(xrMap, r.head.args)
    if len(r.body) > 0 {
        byteCodes = append(byteCodes, ENTER)
    }
    for _, b := range r.body {
        byteCodes = append(byteCodes, compileArgs(xrMap, b.args)...)
        p := proc(b.functor, b.arity())
        i := len(xrMap)
        if v, ok := xrMap[p]; ok {
            i = v
        } else {
            xrMap[p] = i
        }
        byteCodes = append(byteCodes, CALL, instruction(i))
    }
    byteCodes = append(byteCodes, EXIT)
    xr := make(xrTable, len(xrMap))
    for k, v := range xrMap {
        xr[v] = k
    }
    // already done in parsing, so just find highest VAR
    numVars := highestVar(byteCodes)
    return clause{xr, numVars, byteCodes}
}

func compileArgs(xrMap map[entry]int, args []expression) []instruction {
    instrs := []instruction{}
    for _, arg := range args {
        instrs = append(instrs, compileExpression(xrMap, arg)...)
    }
    return instrs
}

func compileExpression(xrMap map[entry]int, e expression) []instruction {
    switch t := e.(type) {
    case variable:
        return []instruction{ VAR, instruction(t) }
    case number:
        n := integer(t)
        i := len(xrMap)
        if v, ok := xrMap[n]; ok {
            i = v
        } else {
            xrMap[n] = i
        }
        return []instruction{ CONST, instruction(i) }
    case symbol:
        a := atom(t)
        i := len(xrMap)
        if v, ok := xrMap[a]; ok {
            i = v
        } else {
            xrMap[a] = i
        }
        return []instruction{ CONST, instruction(i) }
    case process:
        f := functor(t.functor, t.arity())
        i := len(xrMap)
        if v, ok := xrMap[f]; ok {
            i = v
        } else {
            xrMap[f] = i
        }
        instrs := []instruction{FUNCTOR, instruction(i)}
        instrs = append(instrs, compileArgs(xrMap, t.args)...)
        return append(instrs, POP)
    default:
        panic(fmt.Sprintf("unknown type %T", e))
    }
}

// will go berserk if using more vars than fit in an int, but I'll ignore that
func highestVar(b []instruction) int {
    var highest int = -1
    for i:=0; i<len(b); i++ {
        switch b[i] {
        case VAR:
            highest = max(highest, int(b[i+1]))
            i++
        case CALL, CONST, FUNCTOR:
            i++
        }
    }
    return highest + 1
}
