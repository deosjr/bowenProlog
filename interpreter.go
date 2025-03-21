package main

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
    st := state{vc: len(b)}

    // TODO: multiple processes in query
    input := arriveInput{
        p: proc(p[0].functor, p[0].arity()),
        args: p[0].args,
        state: st,
    }
    out := []map[string]expression{}
    for _, ans := range i.arrive(input) {
        m := map[string]expression{}
        for s, v := range b {
            e, ok := ans.sub.get(v)
            if !ok {
                m[s] = v
                continue
            }
            e = ans.sub.walkstar(e)
            m[s] = e
        }
        out = append(out, m)
    }
    return out
}

type state struct {
    sub *substitution
    vo  int // variable offset
    vc  int // variable counter
}

type arriveInput struct {
    p procEntry
    args []expression
    cont []frame
    state state
}

type frame struct {
    pc  []instruction
    xr  xrTable
    vo  int
}

func (i *interpreter) arrive(in arriveInput) []state {
    proc, ok := i.procedures[in.p]
    if !ok {
        return i.arriveBuiltin(in)
    }
    ans := []state{}
    st := in.state
    for _, c := range proc.clauses {
        st.vo = in.state.vc
        st.vc = in.state.vc + c.numVars
        execInput := executeInput{
            pc: c.bytecodes,
            xr: c.xrTable,
            cont: in.cont,
            args: in.args,
            state: st,
        }
        b := i.execute(execInput)
        ans = append(ans, b...)
    }
    return ans
}

func (i *interpreter) arriveBuiltin(in arriveInput) []state {
    // TODO handle builtin call
    execInput := executeInput{
        cont: in.cont,
        state: in.state,
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
    state state
}

// Const/Var/Functor have two modes: matching args (downwards) and creating args (upwards)
// We can tell which mode to operate by looking at args: if there are any, we go downwards
// Otherwise we will build them up in queue. The paper uses difference lists here (!)
// The paper also uses stack both as a stack of lists and as a queue of args!

func (i *interpreter) execute(in executeInput) []state {
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

func (i *interpreter) executeConst(in executeInput) []state {
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
        sub, ok = in.state.sub.unify(in.args[0], number(t))
    case atom:
        sub, ok = in.state.sub.unify(in.args[0], symbol(t))
    default:
        panic("CONST on nonatom")
    }
    if !ok {
        return nil
    }
    in.args = in.args[1:]
    in.state.sub = sub
    return i.execute(in)
}

func (i *interpreter) executeVar(in executeInput) []state {
    if len(in.pc) < 1 {
        panic("VAR without pointer")
    }
    v := variable(in.state.vo + int(in.pc[0]))
    in.pc = in.pc[1:]
    if len(in.args) == 0 {
        in.queue = append(in.queue, v)
        return i.execute(in)
    }
    sub, ok := in.state.sub.unify(in.args[0], v)
    if !ok {
        return nil
    }
    in.args = in.args[1:]
    in.state.sub = sub
    return i.execute(in)
}

func (i *interpreter) executeFunctor(in executeInput) []state {
    if len(in.pc) < 1 {
        panic("FUNCTOR without xr pointer")
    }
    x := in.xr[in.pc[0]].(functorEntry)
    in.pc = in.pc[1:]
    args := make([]expression, x.arity)
    for n:=0; n<x.arity; n++ {
        args[n] = variable(in.state.vc + n)
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
    sub, ok := in.state.sub.unify(in.args[0], p)
    if !ok {
        return nil
    }
    in.stack = append([][]expression{in.args[1:]}, in.stack...) 
    in.args = args
    in.state.sub = sub
    in.state.vc += x.arity
    return i.execute(in)
}

func (i *interpreter) executePop(in executeInput) []state {
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

func (i *interpreter) executeEnter(in executeInput) []state {
    if len(in.args) > 0 || len(in.stack) > 0 {
        return nil  // failure to match, nonempty args/stack
    }
    return i.execute(in)
}

func (i *interpreter) executeCall(in executeInput) []state {
    if len(in.pc) < 1 {
        panic("CALL without xr pointer")
    }
    x := in.xr[in.pc[0]].(procEntry)
    in.pc = in.pc[1:]
    arriveIn := arriveInput{
        p: x,
        args: in.queue,
        cont: append([]frame{{in.pc, in.xr, in.state.vo}}, in.cont...),
        state: in.state,
    }
    return i.arrive(arriveIn)
}

func (i *interpreter) executeExit(in executeInput) []state {
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
        in.state.vo = f.vo
        in.cont = c
        return i.execute(in)
    }
    return []state{in.state}
}
