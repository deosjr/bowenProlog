package main

import (
    "reflect"
    "testing"
)

func TestCompileProcedure(t *testing.T) {
    for i, tt := range []struct{
        rules []rule
        want procedure
    }{
        {
            rules: []rule{
                {head: process{functor: "append", args: []expression{
                    emptylist, variable(0), variable(0),
                }}},
                {head: process{functor: "append", args: []expression{
                    process{functor: "cons", args: []expression{variable(0), variable(1)}},
                    variable(2),
                    process{functor: "cons", args: []expression{variable(0), variable(3)}},
                }},
                body: []process{{functor: "append", args: []expression{
                    variable(1), variable(2), variable(3),
                }}},
                },
            },
            want:  procedure{
                name:   "append",
                arity:  3,
                clauses: []clause{
                    {
                        xrTable: xrTable{constant("nil")},
                        numVars: 1,
                        bytecodes: []instruction{
                            CONST, 0, VAR, 0, VAR, 0, EXIT,
                        },
                    },
                    {
                        xrTable: xrTable{functor("cons", 2), proc("append", 3)},
                        numVars: 4,
                        bytecodes: []instruction{
                            FUNCTOR, 0, VAR, 0, VAR, 1, POP,
                            VAR, 2,
                            FUNCTOR, 0, VAR, 0, VAR, 3, POP,
                            ENTER,
                            VAR, 1, VAR, 2, VAR, 3, CALL, 1,
                            EXIT,
                        },
                    },
                },
            },
        },
    }{
        got := compileProcedure(tt.rules)
        if !reflect.DeepEqual(got, tt.want) {
            t.Errorf("%d: got %q want %q", i, got, tt.want)
        }
    }
}
