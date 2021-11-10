# carrot

carrot is a small programming language has LLVM Backend.

## Requirements

- LLVM 12.x
- clang 12.x

## Install

```zsh
% git clone https://github.com/rb-de0/carrot
% cd carrot
% go install
% go build 
% ./carrot
```

## Spec

### Entry

You need not implement a entry function (like main).
However you need implement return exit code.

```
...
return 0;
```

### Variable

Only Interger Type is implemented.

```
var value = 10;
value = 1;
```

### Control

```
if (x < 10) {
    ...
} else {
    ...
}
```

### Loop

```
for {
    i = i + 1;
    if (i > 10) {
        break;
    }
}
```


### Function

```
fnc sum(x, y) {
    return x + y;
} 
var s = sum(1, 4);
```
