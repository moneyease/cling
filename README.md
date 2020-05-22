# cliNG

This can help you create Command Line Interface for your application. 

CLI structure can be defined in JSON format with very simple rules and run CLI locally or as a server.

This is based on ```readline package``` gives lots of options like password, window resize etc.


## Example

If you need to create CLI like this
```
show customer all [by <region>]
```

Then define JSON like this.

```
{
  "show": {
    "customer": {
      "all": {
        "by": {
          "argGetRegion": {
            "func": "ShowTenant"
          },
          "func": "ShowTenants"
        }
      }
    }
  }
}
```

define three functions

```
var T struct {
... all your stuff
}
```
Input: Ignore it

Output: This will give list of all regions space separated i.e. 'us eu asia'
```
func (t Type) GetRegion   ([]string) string 
```
Input: All arguments are in sequence string sliced i.e. index [0] will have region

Output: All your output or error
```
func (t Type) ShowTenant  ([]string) string 
func (t Type) ShowTenants ([]string) string
```

pass this JSON to create and instance of 

```
```

and start service.
```
```

## Rules
* Few words are reserved 'func' and 'arg' as prefix or whole
* When you hit enter fuction defined with key 'func' will be executed
* JSON keys with 'arg' prefix will give use dynamic options


### Limitations
