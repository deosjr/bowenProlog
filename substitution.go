package main

import "reflect"

func (s *substitution) get(v variable) (expression, bool) {
	return s.Lookup(v)
}

func (s *substitution) put(v variable, e expression) *substitution {
	return s.Insert(v, e)
}

func (s *substitution) walk(u expression) expression {
	uvar, ok := u.(variable)
	if !ok {
		return u
	}
	e, ok := s.get(uvar)
	if !ok {
		return u
	}
	return s.walk(e)
}

func (s *substitution) walkstar(u expression) expression {
	v := s.walk(u)
	switch t := v.(type) {
	case variable:
		return t
	case list:
		return list{head: s.walkstar(t.head), tail: s.walkstar(t.tail)}
    case process:
        args := make([]expression, len(t.args))
        for i:=0; i<len(t.args); i++ {
            args[i] = s.walkstar(t.args[i])
        }
        return process{functor: t.functor, args: args}
	}
	return v
}

func (s *substitution) extend(v variable, e expression) (*substitution, bool) {
	if s.occursCheck(v, e) {
		return nil, false
	}
	return s.put(v, e), true
}

func (s *substitution) unify(u, v expression) (*substitution, bool) {
	u0 := s.walk(u)
	v0 := s.walk(v)
	if reflect.DeepEqual(u0, v0) {
		return s, true
	}
	uvar, uok := u0.(variable)
	if uok {
		return s.extend(uvar, v0)
	}
	vvar, vok := v0.(variable)
	if vok {
		return s.extend(vvar, u0)
	}
	up, uok := u0.(process)
	vp, vok := v0.(process)
	if uok && vok {
        if up.functor != vp.functor || up.arity() != vp.arity() {
            return nil, false
        }
        s0 := s
        for i:=0; i<len(up.args); i++ {
            s, ok := s0.unify(up.args[i], vp.args[i])
            if !ok {
                return nil, false
            }
            s0 = s
        }
		return s0, true
	}
	return nil, false
}

func (s *substitution) occursCheck(v variable, e expression) bool {
	e0 := s.walk(e)
	if evar, ok := e0.(variable); ok {
		return v == evar
	}
	elist, ok := e0.(list)
	if !ok {
		return false
	}
	return s.occursCheck(v, elist.head) || s.occursCheck(v, elist.tail)
}
