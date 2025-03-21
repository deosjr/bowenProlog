package main

import "fmt"

type interpreter struct {
    procedures map[procEntry]procedure
}

func NewInterpreter(procedures []procedure) *interpreter {
    procs := map[procEntry]procedure{}
    for _, p := range procedures {
        procs[proc(p.name, p.arity)] = p
    }
    return &interpreter{procedures: procs}
}

// return all possible bindings. will overflow on infinite answers
// proper support for backtracking would have to make this interactive
// or support run* / runN difference (ie early return)
func (i *interpreter) interpret(s string) []map[string]expression {
    p, b := MustParseProcesses(s)
    // parsing assigned some variables to vars in query
    vc := len(b)
    var substitution *substitution

    // TODO: multiple processes in query
    input := arriveInput{
        p: proc(p[0].functor, p[0].arity()),
        args: p[0].args,
        variablecounter: vc,
        substitution: substitution,
    }
    ans := i.arrive(input)
    if len(ans) == 0 {
        return []map[string]expression{}
    }
    out := map[string]expression{}
    // TODO: multiple possible bindings in answer
    onlyAnswer := ans[0]
    for s, v := range b {
        e, ok := onlyAnswer.Lookup(v)
        if !ok {
            out[s] = v
            continue
        }
        e = onlyAnswer.walk(e)
        out[s] = e
    }
    return []map[string]expression{out}
}

type arriveInput struct {
    p procEntry
    args []expression
    cont []frame
    variablecounter int
    substitution *substitution
}

// TODO: should sub/vc be part of frame?
// should executeInput take a frame too?
type frame struct {
    pc          []instruction
    xr          xrTable
}

func (i *interpreter) arrive(in arriveInput) []*substitution {
    fmt.Println("ARRIVE", in)
    proc, ok := i.procedures[in.p]
    if !ok {
        return i.arriveBuiltin(in)
    }
    ans := []*substitution{}
    offset := in.variablecounter
    for _, c := range proc.clauses {
        execInput := executeInput{
            pc: c.bytecodes,
            xr: c.xrTable,
            cont: in.cont,
            args: in.args,
            varOffset: offset,
            // TODO: varcounter needs to take into account fresh vars
            variablecounter: in.variablecounter + c.numVars + 100,
            substitution: in.substitution,
        }
        // TODO: offset needs to take into account fresh vars minted in execution
        offset += c.numVars
        b := i.execute(execInput)
        ans = append(ans, b...)
    }
    return ans
}

func (i *interpreter) arriveBuiltin(in arriveInput) []*substitution {
    // TODO handle builtin call
    execInput := executeInput{
        cont: in.cont,
        variablecounter: in.variablecounter,
        substitution: in.substitution,
    }
    return i.executeExit(execInput)
}

type executeInput struct {
    pc []instruction
    xr xrTable
    cont []frame
    args []expression
    stack [][]expression
    queue []expression
    varOffset int
    variablecounter int
    substitution *substitution
}

// Const/Var/Functor have two modes: matching args (downwards) and creating args (upwards)
// We can tell which mode to operate by looking at args: if there are any, we go downwards
// Otherwise we will build them up in queue. The paper uses difference lists here (!)
// The paper also uses stack both as a stack of lists and as a queue of args!

func (i *interpreter) execute(in executeInput) []*substitution {
    fmt.Println("EXECUTE", in)
    if len(in.pc) < 1 {
        panic("executing empty instruction list")
    }
    ins, pc := in.pc[0], in.pc[1:]
    in.pc = pc
    switch ins {
    case CONST:
        return i.executeConst(in)
    case VAR:
        return i.executeVar(in)
    case FUNCTOR:
        return i.executeFunctor(in)
    case POP:
        return i.executePop(in)
    case ENTER:
        return i.executeEnter(in)
    case CALL:
        return i.executeCall(in)
    case EXIT:
        return i.executeExit(in)
    default:
        panic("unknown instruction")
    }
    return nil
}

func (i *interpreter) executeConst(in executeInput) []*substitution {
    if len(in.pc) < 1 {
        panic("CONST without xr pointer")
    }
    x := in.xr[in.pc[0]]
    in.pc = in.pc[1:]
    // TODO: this kind of typecasting is inefficient and should be removed
    if len(in.args) == 0 {
        switch t := x.(type) {
        case integer:
            in.queue = append(in.queue, number(t))
        case atom:
            in.queue = append(in.queue, symbol(t))
        default:
            panic("CONST on nonatom")
        }
        return i.execute(in)
    }
    var sub *substitution
    var ok bool
    switch t := x.(type) {
    case integer:
        sub, ok = in.substitution.unify(in.args[0], number(t))
    case atom:
        sub, ok = in.substitution.unify(in.args[0], symbol(t))
    default:
        panic("CONST on nonatom")
    }
    if !ok {
        return nil
    }
    in.args = in.args[1:]
    in.substitution = sub
    return i.execute(in)
}

func (i *interpreter) executeVar(in executeInput) []*substitution {
    if len(in.pc) < 1 {
        panic("VAR without pointer")
    }
    v := variable(in.varOffset + int(in.pc[0]))
    in.pc = in.pc[1:]
    if len(in.args) == 0 {
        in.queue = append(in.queue, v)
        return i.execute(in)
    }
    sub, ok := in.substitution.unify(in.args[0], v)
    if !ok {
        return nil
    }
    in.args = in.args[1:]
    in.substitution = sub
    return i.execute(in)
}

func (i *interpreter) executeFunctor(in executeInput) []*substitution {
    if len(in.pc) < 1 {
        panic("FUNCTOR without xr pointer")
    }
    x := in.xr[in.pc[0]].(functorEntry)
    in.pc = in.pc[1:]
    args := make([]expression, x.arity)
    for n:=0; n<x.arity; n++ {
        args[n] = variable(in.variablecounter + n)
    }
    p := process{
        functor: x.name,
        args:    args,
    }
    if len(in.args) == 0 {
        in.queue = append(in.queue, p)
        in.stack = append([][]expression{in.args[1:]}, in.stack...) 
        in.args = args
        return i.execute(in)
    }
    sub, ok := in.substitution.unify(in.args[0], p)
    if !ok {
        return nil
    }
    in.stack = append([][]expression{in.args[1:]}, in.stack...) 
    in.args = args
    in.substitution = sub
    in.variablecounter += x.arity
    return i.execute(in)
}

func (i *interpreter) executePop(in executeInput) []*substitution {
    if len(in.args) > 0 {
        panic("POP with nonempty args")
    }
    if len(in.stack) == 0 {
        panic("POP with empty stack")
    }
    in.args = in.stack[0]
    in.stack = in.stack[1:]
    return i.execute(in)
}

func (i *interpreter) executeEnter(in executeInput) []*substitution {
    if len(in.args) > 0 || len(in.stack) > 0 {
        return nil  // failure to match, nonempty args/stack
    }
    return i.execute(in)
}

func (i *interpreter) executeCall(in executeInput) []*substitution {
    if len(in.pc) < 1 {
        panic("CALL without xr pointer")
    }
    x := in.xr[in.pc[0]].(procEntry)
    in.pc = in.pc[1:]
    arriveIn := arriveInput{
        p: x,
        args: in.queue,
        cont: append([]frame{{in.pc, in.xr}}, in.cont...),
        variablecounter: in.variablecounter,
        substitution: in.substitution,
    }
    return i.arrive(arriveIn)
}

func (i *interpreter) executeExit(in executeInput) []*substitution {
    if len(in.pc) > 0 {
        panic("EXIT on nonempty instruction list")
    }
    if len(in.args) > 0 || len(in.stack) > 0 {
        return nil  // failure to match, nonempty args/stack
    }
    if len(in.cont) > 0 {
        f, c := in.cont[0], in.cont[1:]
        in.pc = f.pc
        in.xr = f.xr
        in.cont = c
        return i.execute(in)
    }
    return []*substitution{in.substitution}
}
