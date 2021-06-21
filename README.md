# Act

Act is a task runner and supervisor written in Go which aims to provide the following features:

* process supervision in a project level
* allow tasks to be written as simple bash scripting if wanted
* sub tasks which can be invoked like `act run cmd1 cmd2`
* dynamically include tasks defined in other locations allowing spliting task definition
* regex match task names

## Instalation

@TODO : We need to compile binaries and have a nice way to install act like we have for volta https://volta.sh/.

### From Source

First you need to have go >= 1.16 installed in your machine. Then after cloning this repo you can build act binary by doing:

```bash
cd /path/to/act/folder

GOROOT=/usr/local/bin go install github.com/nosebit/act
```

Feel free to change GOROOT to whatever destination you want.

## How to Use

After installing `act` we create an `actfile.yml` file in root dir of our project so we can describe the acts we can run like the following:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is a simple hello world act example
    cmds:
      - echo "Hello foo"
```

and then we run the act with the following command:

```bash
act run foo
```

If we don't specify `cmds` in `actfile.yml` file for `foo` then act going to look up for a shell script file named `foo.sh` in `acts` folder and if that script file is found then act going to run it which allows very powerful acts. In other words we can have the following folder structure:

```
+ my-project
|--+ acts
   |-- foo.sh
|-- actfile.yml
```

where actfile is defined like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is a simple hello world act example
```

since we don't define `cmds` in foo act then when executing `act run foo` we going to execute `acts/foo.sh` script.

### Long Running Acts

If an act is written to be a long running process like the following:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is a simple long running act example
    cmds:
      - while true; echo "Hello long running"; sleep 5; done
```

then we can run it as a daemon using the following command:

```bash
act run -d foo
```

To list all running acts we can use:

```bash
act list
```

and finally to stop an act we can use:

```bash
act stop foo
```

### Subacts

We can defined subacts like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is a simple long running act example
    acts:
      bar:
        cmds:
          - echo "im bar subact of foo"
```

which we can run like this:

```bash
act run foo.bar
```

### Act Name Matching

The act name we use in `actfile.yml` is actually a regex we going to match against the name use provide to `act run` command. That way if we have:

```yaml
# actfile.yml
version: 1

acts:
  foo-.+:
    desc: This is a generic foo act.
    cmds:
      - echo "i'm $ACT_NAME"
```

we can call

```bash
act run foo-bar
```

and see `i'm foo-bar` printed in the screen. 

### Including Acts

We can even include acts from another actfile as subacts in the following way:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is an act that include subacts.
    include: another/actfile.yml
```

```yaml
# another/actfile.yml
version: 1

acts:
  foo:
    desc: This is a sample act.
    cmds:
      - echo "im bar"
```

and then we can call

```bash
act run foo.bar
```

Inclusion and act name matching allow us to do some interesting things like this:

```yaml
# actfile.yml
version: 1

acts:
  .+:
    desc: Scope acts from a sub directory.
    include: "{{.actName}}/actfile.yml"
```

and then in a subdirectory called `backend` for example we can have:

```yaml
# backend/acfile.yml
version: 1

acts:
  start:
    desc: Start backend service written in nodejs.
    cmds:
      - node index.js
```

This way we can start the backend from the root project directory by running:

```bash
act run backend.start
```

### Act Called From Command

We can call another act from on act command like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is an act that include subacts.
    cmds:
      - echo "foo before bar"
      - act: bar
      - echo "foo after bar"
  bar:
    cmds:
      - echo "im bar"
```

and then when we call `act run foo` we going to see

```bash
foo before bar
im bar
foo after bar
```
