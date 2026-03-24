# Specification Templates

## Function Specification Template

```markdown
## Specification: `function_name`

### Signature
```go
func FunctionName(param1 Type1, param2 Type2) (ReturnType, error)
```

### Purpose
[Brief description of what the function accomplishes]

### Parameters
- `param1`: `Type1` - [Description]
- `param2`: `Type2` - [Description]

### Return Value
- `ReturnType`: [Description of what is returned]
- `error`: [Error conditions]

### Preconditions
- [List any required conditions]
- [State requirements before calling]

### Postconditions
- [Guarantees after successful execution]
- [State changes made by function]

### Side Effects
- [I/O operations performed]
- [Mutations to shared state]
- [External system interactions]

### Error Handling
- Returns error when: [conditions]
- Error types: [specific errors]

### Thread Safety
- [Safe/unsafe for concurrent use]
- [Locks or synchronization used]

### Example
```go
result, err := FunctionName(value1, value2)
if err != nil {
    // handle error
}
// use result
```
```

## Struct/Class Specification Template

```markdown
## Specification: `StructName`

### Purpose
[Brief description of the struct's role]

### Fields
- `Field1`: `Type` - [Description]
- `Field2`: `Type` - [Description]

### Methods
- `Method1()`: [Description]
- `Method2()`: [Description]

### Invariants
- [Conditions that are always true]
- [Constraints on field values]

### Usage Pattern
```go
s := StructName{
    Field1: value1,
    Field2: value2,
}
s.Method1()
```
```

## Interface Specification Template

```markdown
## Specification: `InterfaceName`

### Purpose
[What this interface abstracts]

### Methods
#### `MethodSignature`
[Description of method's contract]

### Implementations
Known implementations:
- `ConcreteType1`: [Brief description]
- `ConcreteType2`: [Brief description]

### Usage Example
```go
var iface InterfaceName = &ConcreteType1{}
iface.Method()
```
```
