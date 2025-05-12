| Name                 | Type          | Previous | Input     | Default | Options           | Validation | -> | Output | Optional | ErrorCreate     |
|----------------------|---------------|----------|-----------|---------|-------------------|------------|----|--------|----------|-----------------|
|                      | Empty Vals    |          |           |         |                   |            |    |        |          |                 |
| Empty                | string,number |          |           |         |                   |            |    | ""     | false    |                 |
| EmptyDupeOps         | string,number |          |           |         | 1,1,1             |            |    |        |          | unique          |
| EmptyList            | list(string)  |          |           |         |                   |            |    | ""     | false    |                 |
| EmptyListDupeOpts    | list(string)  |          |           |         | ["a"],["a"]       |            |    |        |          | unique          |
| EmptyMulti           | tag-select    |          |           |         |                   |            |    | ""     | false    |                 |
| EmptyOpts            | string,number |          |           |         | 1,2,3             |            |    | ""     | false    |                 |
| EmptyRegex           | string        |          |           |         |                   | world      |    |        |          | regex error     |
| EmptyMin             | number        |          |           |         |                   | 1-10       |    |        |          | 1 <  < 10       |
| EmptyMinOpt          | number        |          |           |         | 1,2,3             | 2-5        |    |        |          | valid option    |
| EmptyRegexOpt        | string        |          |           |         | "hello","goodbye" | goodbye    |    |        |          | valid option    |
| EmptyRegexOk         | string        |          |           |         |                   | .*         |    | ""     | false    |                 |
| EmptyInc             | number        | 4        |           |         |                   | increasing |    |        |          | monotonicity    |
| EmptyDec             | number        | 4        |           |         |                   | decreasing |    |        |          | monotonicity    |
|                      |               |          |           |         |                   |            |    |        |          |                 |
|                      | Default Set   |          | No inputs |         |                   |            |    |        |          |                 |
| NumDef               | number        |          |           | 5       |                   |            |    | 5      | true     |                 |
| NumDefVal            | number        |          |           | 5       |                   | 3-7        |    | 5      | true     |                 |
| NumDefInv            | number        |          |           | 5       |                   | 10-        |    |        |          | 10 < 5 < 0      |
| NumDefOpts           | number        |          |           | 5       | 1,3,5,7           | 2-6        |    | 5      | true     |                 |
| NumDefNotOpts        | number        |          |           | 5       | 1,3,7,9           | 2-6        |    |        |          | valid option    |
| NumDefInvOpt         | number        |          |           | 5       | 1,3,5,7           | 6-10       |    |        |          | 6 < 5 < 10      |
| NumDefNotNum         | number        |          |           | a       |                   |            |    |        |          | type "number"   |
| NumDefOptsNotNum     | number        |          |           | 1       | 1,a,2             |            |    |        |          | type "number"   |
| NumDefInc            | number        | 4        |           | 5       |                   | increasing |    | 5      | true     |                 |
| NumDefIncBad         | number        | 6        |           | 5       |                   | increasing |    |        |          | greater         |
| NumDefDec            | number        | 6        |           | 5       |                   | decreasing |    | 5      | true     |                 |
| NumDefDecBad         | number        | 4        |           | 5       |                   | decreasing |    |        |          | lower           |
| NumDefDecEq          | number        | 5        |           | 5       |                   | decreasing |    | 5      | true     |                 |
| NumDefIncEq          | number        | 5        |           | 5       |                   | increasing |    | 5      | true     |                 |
| NumDefIncNaN         | number        | a        |           | 5       |                   | increasing |    | 5      | true     |                 |
| NumDefDecNaN         | number        | b        |           | 5       |                   | decreasing |    | 5      | true     |                 |
|                      |               |          |           |         |                   |            |    |        |          |                 |
| StrDef               | string        |          |           | hello   |                   |            |    | hello  | true     |                 |
| StrMonotonicity      | string        |          |           | hello   |                   | increasing |    |        |          | monotonic       |
| StrDefInv            | string        |          |           | hello   |                   | world      |    |        |          | regex error     |
| StrDefOpts           | string        |          |           | a       | a,b,c             |            |    | a      | true     |                 |
| StrDefNotOpts        | string        |          |           | a       | b,c,d             |            |    |        |          | valid option    |
| StrDefValOpts        | string        |          |           | a       | a,b,c,d,e,f       | [a-c]      |    | a      | true     |                 |
| StrDefInvOpt         | string        |          |           | d       | a,b,c,d,e,f       | [a-c]      |    |        |          | regex error     |
|                      |               |          |           |         |                   |            |    |        |          |                 |
| LStrDef              | list(string)  |          |           | ["a"]   |                   |            |    | ["a"]  | true     |                 |
| LStrDefOpts          | list(string)  |          |           | ["a"]   | ["a"], ["b"]      |            |    | ["a"]  | true     |                 |
| LStrDefNotOpts       | list(string)  |          |           | ["a"]   | ["b"], ["c"]      |            |    |        |          | valid option    |
|                      |               |          |           |         |                   |            |    |        |          |                 |
| MulDef               | tag-select    |          |           | ["a"]   |                   |            |    | ["a"]  | true     |                 |
| MulDefOpts           | multi-select  |          |           | ["a"]   | a,b               |            |    | ["a"]  | true     |                 |
| MulDefNotOpts        | multi-select  |          |           | ["a"]   | b,c               |            |    |        |          | valid option    |
|                      |               |          |           |         |                   |            |    |        |          |                 |
|                      | Input Vals    |          |           |         |                   |            |    |        |          |                 |
| NumIns               | number        |          | 3         |         |                   |            |    | 3      | false    |                 |
| NumInsOptsNaN        | number        |          | 3         | 5       | a,1,2,3,4,5       | 1-3        |    |        |          | type "number"   |
| NumInsNotNum         | number        |          | a         |         |                   |            |    |        |          | type "number"   |
| NumInsNotNumInv      | number        |          | a         |         |                   | 1-3        |    |        |          | 1 < a < 3       |
| NumInsDef            | number        |          | 3         | 5       |                   |            |    | 3      | true     |                 |
| NumIns/DefInv        | number        |          | 3         | 5       |                   | 1-3        |    | 3      | true     |                 |
| NumIns=DefInv        | number        |          | 5         | 5       |                   | 1-3        |    |        |          | 1 < 5 < 3       |
| NumInsOpts           | number        |          | 3         | 5       | 1,2,3,4,5         | 1-3        |    | 3      | true     |                 |
| NumInsNotOptsVal     | number        |          | 3         | 5       | 1,2,4,5           | 1-3        |    |        |          | valid option    |
| NumInsNotOptsInv     | number        |          | 3         | 5       | 1,2,4,5           | 1-2        |    |        | true     | valid option    |
| NumInsNotOpts        | number        |          | 3         | 5       | 1,2,4,5           |            |    |        |          | valid option    |
| NumInsNotOpts/NoDef  | number        |          | 3         |         | 1,2,4,5           |            |    |        |          | valid option    |
| NumInsInc            | number        | 4        | 5         | 3       |                   | increasing |    | 5      | true     |                 |
| NumInsIncBad         | number        | 6        | 5         | 7       |                   | increasing |    |        |          | greater         |
| NumInsDec            | number        | 6        | 5         | 7       |                   | decreasing |    | 5      | true     |                 |
| NumInsDecBad         | number        | 4        | 5         | 3       |                   | decreasing |    |        |          | lower           |
| NumInsDecEq          | number        | 5        | 5         | 5       |                   | decreasing |    | 5      | true     |                 |
| NumInsIncEq          | number        | 5        | 5         | 5       |                   | increasing |    | 5      | true     |                 |
|                      |               |          |           |         |                   |            |    |        |          |                 |
| StrIns               | string        |          | c         |         |                   |            |    | c      | false    |                 |
| StrInsDupeOpts       | string        |          | c         |         | a,b,c,c           |            |    |        |          | unique          |
| StrInsDef            | string        |          | c         | e       |                   |            |    | c      | true     |                 |
| StrIns/DefInv        | string        |          | c         | e       |                   | [a-c]      |    | c      | true     |                 |
| StrIns=DefInv        | string        |          | e         | e       |                   | [a-c]      |    |        |          | regex error     |
| StrInsOpts           | string        |          | c         | e       | a,b,c,d,e         | [a-c]      |    | c      | true     |                 |
| StrInsNotOptsVal     | string        |          | c         | e       | a,b,d,e           | [a-c]      |    |        |          | valid option    |
| StrInsNotOptsInv     | string        |          | c         | e       | a,b,d,e           | [a-b]      |    |        |          | valid option    |
| StrInsNotOpts        | string        |          | c         | e       | a,b,d,e           |            |    |        |          | valid option    |
| StrInsNotOpts/NoDef  | string        |          | c         |         | a,b,d,e           |            |    |        |          | valid option    |
| StrInsBadVal         | string        |          | c         |         | a,b,c,d,e         | 1-10       |    |        |          | min cannot      |
|                      |               |          |           |         |                   |            |    |        |          |                 |
|                      | list(string)  |          |           |         |                   |            |    |        |          |                 |
| LStrIns              | list(string)  |          | ["c"]     |         |                   |            |    | ["c"]  | false    |                 |
| LStrInsNotList       | list(string)  |          | c         |         |                   |            |    |        |          | list of strings |
| LStrInsDef           | list(string)  |          | ["c"]     | ["e"]   |                   |            |    | ["c"]  | true     |                 |
| LStrIns/DefInv       | list(string)  |          | ["c"]     | ["e"]   |                   | [a-c]      |    |        |          | regex cannot    |
| LStrInsOpts          | list(string)  |          | ["c"]     | ["e"]   | ["c"],["d"],["e"] |            |    | ["c"]  | true     |                 |
| LStrInsNotOpts       | list(string)  |          | ["c"]     | ["e"]   | ["d"],["e"]       |            |    |        |          | valid option    |
| LStrInsNotOpts/NoDef | list(string)  |          | ["c"]     |         | ["d"],["e"]       |            |    |        |          | valid option    |
|                      |               |          |           |         |                   |            |    |        |          |                 |
| MulInsOpts           | multi-select  |          | ["c"]     | ["e"]   | c,d,e             |            |    | ["c"]  | true     |                 |
| MulInsNotListOpts    | multi-select  |          | c         | ["e"]   | c,d,e             |            |    |        |          | json encoded    |
| MulInsNotOpts        | multi-select  |          | ["c"]     | ["e"]   | d,e               |            |    |        |          | valid option    |
| MulInsNotOpts/NoDef  | multi-select  |          | ["c"]     |         | d,e               |            |    |        |          | valid option    |
| MulInsInvOpts        | multi-select  |          | ["c"]     | ["e"]   | c,d,e             | [a-c]      |    |        |          | regex cannot    |