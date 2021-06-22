# Act

Act is a task runner and supervisor written in Go which aims to provide the following features:

* process supervision in a project level
* allow tasks to be written as simple bash scripting if wanted
* sub tasks which can be invoked with `act run foo.bar` where `bar` is a sub task of `foo` task (tasks are called acts here)
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

If you need to specify a more powerful command you can use shell scripting directly in `cmds` field like this:

```yaml
# actfile.yml
version: 1

acts:
  build-deps:
    desc: This act going to build dependencies in a workspace and optionally clean everything before start.
    cmds: |
      echo "args=$@"

      for i in "$@"; do
        case $i in
          --clean)
          CLEAN="true"
          shift # past argument=value
          ;;
        esac
      done

      if [ -n "$CLEAN" ]; then
        echo "Running clean mode"
        # First let's remove all sub node_modules
        find . -name "node_modules" -type d -prune -exec rm -rf '{}' +
      fi

      # Now install dependencies
      #
      # We put yarn install inside a loop to get rid of an ENOENT error we were getting.
      # Check this out: https://github.com/yarnpkg/yarn/issues/2629
      yarn install --ignore-engines $@

```

Notice that acts can receive command line arguments which are being used in `build-deps` command via `$@`. In this case we are even allowing a cli flag `--clean` which allows us to run `act run deps --clean`.

If we don't want to "polute" the actfile with a lot of scripting like we did for `build-deps` we could remove `cmds` field entirely and add a `build-deps` script inside an `acts` folder at the root of our project directory. When running `act run build-deps` act binary going to see the act does not have any `cmds` defined and going to look up a `acts/build-deps.sh` script to run instead. In other words, for a generic act called `foo` the folder structure should look like this:

```
+ my-project
|--+ acts
   |-- foo.sh
|-- actfile.yml
```

where actfile should be defined like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is a simple hello world act example
```

since we don't define `cmds` in foo act then when executing `act run foo` we going to execute `acts/foo.sh` script.

### Before Commands

If we need to run commands before any act executed we can do it like this:

```yaml
# actfile.yml
version: 1

before:
  - echo "running before"
  - act: bar inline-arg-1 inline-arg-2
  - act: zoo
    args:
      - non-inline-arg-1
      - non-inline-arg-2

acts:
  foo:
    cmds: echo "im foo"
  bar:
    cmds: echo "im bar with args=$@"
  zoo:
    cmds: echo "im zoo with args=$@"

```

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
