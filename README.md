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

If we need to specify a diferent actfile to be used we can do it like this:

```bash
act run -f=/path/to/actfile.yml foo
```

If we need to specify a more powerful command we can use shell scripting directly in `cmds` field like this:

```yaml
# actfile.yml
version: 1

acts:
  build-deps:
    desc: This act going to build dependencies in a workspace and optionally clean everything before start.
    cmds: |
      echo "args=$@"
      yarn install --ignore-engines $@

```

Notice that acts can receive command line arguments which are being used in `build-deps` command via `$@`.

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

before-all:
  cmds:
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


### Commands Parallel Execution

By default Act going to run commands in sequence but if we want commands to be executed in parallel we can do like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    parallel: true
    cmds:
      - |
        echo "running cmd1"

        for i in {1..5}; do
          echo "cmd1 $i"; sleep 2
        done
      - |
        echo "running cmd2"

        for i in {1..5}; do
          echo "cmd2 $i"; sleep 2
        done
```


### Sharing Env Vars Between Commands

Act going to run commands in independent shell environments to allow parallel execution as discussed in the previous section. That way if we need to share variables between commands (or acts) we can write variables as `key=val` strings to a special dotenv file which location is provided by `$ACT_ENV` var. Here an example:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    cmds:
      - echo "MY_VAR=nosebit" >> $ACT_ENV
      - echo "MY_VAR is $MY_VAR"
```

This way when we run `act run foo` we should see `MY_VAR is nosebit` printed to the screen.

**WARNING**: Be careful with race conditions when reading/write variables to `$ACT_ENV` when running commands in parallel.


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
    include: "{{.ActName}}/actfile.yml"
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

### Redirect Act Call To Another Actfile

If we need to redirect the call to a `foo` act to an act with same name in another actfile we can use it like this:

```yaml
# acfile.yml
version: 1

acts:
  foo:
    redirect: another/actfile.yml
```

and

```yaml
# another/actfile.yml
version: 1

acts:
  foo:
    cmds: echo "im foo in another/actfile.yml"
```

This way when we call `act run foo` in the folder containing `actfile.yml` we going to see `im foo in another/actfile.yml` printed to the screen. When used with regex name matching feature of Act this can be very powerful because we can redirect a group of call to another actfile. Suppose we have the following folder structure:

```txt
my-worspace
  |-- backend
  |   |-- actfile.yml
  |-- frontned
  |   |-- actfile.yml
  |-- actfile.yml
```

with following actfile for backend:

```yaml
# backend/actfile.yml
version: 1

acts:
  start:
    cmds: node index.js
  build:
    cmds: echo "lets transpile ts to js"
```

We can redirect all act calls to reach backend acts by default like this:

```yaml
# actfile.yml
version: 1

acts:
  .+:
    redirect: backend/actfile.yml
```

This way if we run `act run start` we going to start backend service.


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

### Variables

We have some variables at our disposition like the following:

  * `ActName` : The matched act name.
  * `ActFilePath`: The full actfile name which is being used (where act were matched).
  * `ActFileDir`: The base directory path of the actfile.
  * `ActEnv`: Path to a runtime env file which can be used in commands to share variables at execution time.

@TODO : We need to allow user specifying variables directly in actfile.

### Loading Variables From File

Act support dotenv vars file to be loaded before the execution of an act. So suppose we have the following `.vars` file in the root of our project:

```txt
MY_VAR=my-val
```

then we can load this variable like this:

```yaml
# actfile.yml
version: 1

envfile: .vars

acts:
  foo:
    cmds:
      - echo "my rendered var is {{.MY_VAR}}"
      - echo "my env var is $MY_VAR"
```

and we can even use those loaded env vars in other fields like include/from via go template language like this:

```yaml
# actfile.yml
version: 1

env: .vars

acts:
  foo:
    include: "{{.MY_VAR}}/actfile.yml"
```

### Command Line Flags

If we want to support command line flags in our acts we can do it like the following:

```yaml
# actfile.yml
version: 1

env: .vars

acts:
  foo:
    flags:
      - daemon:false
      - name
    cmds:
      - echo "daemon boolean flag => $FLAG_DAEMON"
      - echo "name string flag => $FLAG_NAME"
      - echo "other args are => $@"
```

This way we can run `act run foo -daemon -name=Bruno arg1 arg2`. Note that boolean flags which can be provided without values should have a default `false` added.


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

and finally to stop an act by it's name we can use:

```bash
act stop foo
```

If we want to run multiple acts as separate processes to be self managed we can do like this:

```yaml
# actfile.yml
version: 1

acts:
  long1:
    cmds: while true; do echo "hello long1"; sleep 4; done
  long2:
    cmds: while true; do echo "hello long2"; sleep 2; done

  all:
    parallel: true
    cmds:
      - act run -f={{.ActFilePath}} long1
      - act run -f={{.ActFilePath}} long2
```

This way if we run `act run all` we going to run long1 and long2 as different act processes and we can stop only one of those with `act stop long1` for example. If we want to kill everything we can do `act stop all`.


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
