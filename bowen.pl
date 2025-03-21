% D.L. Bowen, L.M. Byrd, W.F. Clocksin - A Portable Prolog Compiler

run :-
    Compiled = procedure( append/3, [
        clause( xrtable(nil), 1,
            [ const, 1,                         % nil
              var,   1,                         % L
              var,   1,                         % L
              exit]),
        clause( xrtable(cons/2, procedure(append/3)), 4,
            [ functor, 1, var, 1, var, 2, pop,  % cons(X, L1)
              var, 3,                           % L2
              functor, 1, var, 1, var, 4, pop,  % cons(X, L3)
              enter,
              var, 2, var, 3, var, 4, call, 2,  % append(L1, L2, L3)
              exit])]),
    assertz(Compiled),
    arrive(append/3,[cons(a, cons(b,nil)), cons(c,nil), L],[]),
    writeln(L). % expect L = cons(a, cons(b, cons(c,nil))).

arrive(Proc, Args, Cont) :-
    procedure(Proc, Clauses), !,                % Find clause list for Proc
    member(clause(XR, NVars, PC), Clauses),     % Select one
    functor(Vars, vars, NVars),                 % Make new set of variables
    execute(PC, XR, Vars, Cont, Args, []).      % Go to execute byte-codes
arrive(Name/_Arity, Args, Cont) :-
    % TODO (omitted from paper): check arity against len args
    Proc =.. [Name|Args],                       % No compiled clauses: call
    call(Proc),                                 %   normal Prolog procedure
    execute([exit],_,_,Cont,_,_).               % and continue

execute([const, X|PC], XR, Vars, Cont, [Arg|Arest], Astack) :- !,
    arg(X, XR, Arg),                            % Match XR entry with Arg
    execute(PC, XR, Vars, Cont, Arest, Astack).
execute([var, V|PC], XR, Vars, Cont, [Arg|Arest], Astack) :- !,
    arg(V, Vars, Arg),                          % Match variable with Arg
    execute(PC, XR, Vars, Cont, Arest, Astack).
execute([functor, X|PC], XR, Vars, Cont, [Arg|Arest], Astack) :- !,
    arg(X, XR, Fatom/Farity),                   % Get functor from XR table
    functor(Arg, Fatom, Farity),                % Match principal functors
    Arg =.. [Fatom|Args],                       % Get Args of Arg term
    execute(PC, XR, Vars, Cont, Args, [Arest|Astack]).
execute([pop|PC], XR, Vars, Cont, [], [Args|Astack]) :- !,  % Pop Args off Astack
    execute(PC, XR, Vars, Cont, Args, Astack).
execute([enter|PC], XR, Vars, Cont, [], []) :- !,
    execute(PC, XR, Vars, Cont, Args, Args).    % Initialise diff list:
execute([call, X|PC], XR, Vars, Cont, [], Args) :- !,
    arg(X, XR, procedure(Proc)),                % Extract proc name from XR
    arrive(Proc, Args, [frame(PC, XR, Vars)|Cont]).         % Save context & go
execute([exit], _, _, [frame(PC, XR, Vars)|Cont], [], []) :- !,
    execute(PC, XR, Vars, Cont, Args, Args).    % Resume previous context
execute([exit], _, _, [], [], []) :- !.         % No previous context: stop
