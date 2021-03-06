# Act

Act is a task runner and supervisor tool written in Go which aims to provide the following features:

* process supervision in a project level
* allow tasks to be written as simple bash scripting if wanted
* sub tasks which can be invoked with `act run foo.bar` where `bar` is a sub task of `foo` task (tasks are called acts here)
* dynamically include tasks defined in other locations allowing spliting task definition
* regex match task names
* and much more :)


## Installation

@TODO : We need to compile binaries and have a nice way to install act like we have for volta https://volta.sh/.

### Download Binary

**Linux**

```bash
wget -q -O - https://github.com/nosebit/act/releases/download/v1.5.3/act-1.5.3-linux-amd64.tar.gz | sudo tar -xzf - -C /usr/local/bin
```

**MacOS**

```bash
wget -q -O - https://github.com/nosebit/act/releases/download/v1.5.3/act-1.5.3-darwin-amd64.tar.gz | sudo tar -xzf - -C /usr/local/bin
```

### From Source

First you need to have go >= 1.16 installed in your machine. Then after cloning this repo you can build act binary by doing:

```bash
cd /path/to/act/folder

GOROOT=/usr/local/bin go install ./...
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
    start:
      - echo "Hello foo"
```

and then we run the act with the following command:

```bash
act run foo
```

or shortly we can do just:

```bash
actr foo
```

If we need to specify a different actfile to be used we can do it like this:

```bash
act run -f=/path/to/actfile.yml foo
```

If we need to specify a more powerful command we can use shell scripting directly in `start` field like this:

```yaml
# actfile.yml
version: 1

acts:
  build-deps:
    desc: This act going to build dependencies in a workspace and optionally clean everything before start.
    start: |
      echo "args=$@"
      yarn install --ignore-engines $@

```

Notice that acts can receive command line arguments which are being used in `build-deps` command via `$@`.


### Running Scripts as Commands

If we don't want to "pollute" the actfile with a lot of scripting like we did for `build-deps` we can provide a script file using the `script` field of a command like this:

```yaml
# actfile.yml
version: 1

acts:
  build-deps:
    desc: This act going to build dependencies in a workspace and optionally clean everything before start.
    start:
      - script: /path/to/script.sh

```

By default Act going to use `bash` as it's default shell but you can customize the shell to use via `shell` field in actfile, act or command levels like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    start:
      - cmd: echo "hello"
        shell: sh
      - script: /path/to/script.sh
        shell: bash # default
  bar:
    shell: bash # default
    start:
      - echo "bar 1"
      - echo "bar 2"
```


### Before Commands

If we need to run commands before any act executed we can do it like this:

```yaml
# actfile.yml
version: 1

before-all:
  start:
    - echo "running before"
    - act: bar inline-arg-1 inline-arg-2
    - act: zoo
      args:
        - non-inline-arg-1
        - non-inline-arg-2

acts:
  foo:
    start: echo "im foo"
  bar:
    start: echo "im bar with args=$@"
  zoo:
    start: echo "im zoo with args=$@"
```


### Commands Parallel Execution

By default Act going to run commands in sequence but if we want commands to be executed in parallel we can do like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    start:
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


### Act Name Matching

The act name we use in `actfile.yml` is actually a regex we going to match against the name use provide to `act run` command. That way if we have:

```yaml
# actfile.yml
version: 1

acts:
  foo-.+:
    desc: This is a generic foo act.
    start:
      - echo "i'm $ACT_NAME"
```

we can call

```bash
act run foo-bar
```

and see `i'm foo-bar` printed in the screen.


### Subacts

We can defined subacts like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: Act with subacts.
    acts:
      bar:
        start:
          - echo "im bar subact of foo"
```

which we can run like this:

```bash
act run foo.bar
```
A special index subact named `_` can be provided to match the parent act name like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: Act with subacts and index subact.
    acts:
      _:
        start: echo "im foo"
      bar:
        start: echo "im bar subact of foo"
```

Now we can run `act run foo` to see `im foo` printde to the screen and `act run foo.bar` to see `im bar subact of foo`.

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
    start:
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
# backend/actfile.yml
version: 1

acts:
  up:
    desc: Start backend service written in nodejs.
    start:
      - node index.js
```

This way we can start the backend from the root project directory by running:

```bash
act run backend.up
```

### Redirect Act Call To Another Actfile

If we need to redirect the call to a `foo` act to an act with same name in another actfile we can use it like this:

```yaml
# actfile.yml
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
    start: echo "im foo in another/actfile.yml"
```

This way when we call `act run foo` in the folder containing `actfile.yml` we going to see `im foo in another/actfile.yml` printed to the screen. When used with regex name matching feature of Act this can be very powerful because we can redirect a group of call to another actfile. Suppose we have the following folder structure:

```txt
my-workspace
  |-- backend
  |   |-- actfile.yml
  |-- frontend
  |   |-- actfile.yml
  |-- actfile.yml
```

with following actfile for backend:

```yaml
# backend/actfile.yml
version: 1

acts:
  up:
    start: node index.js
  build:
    start: echo "lets transpile ts to js"
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
    start:
      - echo "foo before bar"
      - act: bar
      - echo "foo after bar"
  bar:
    start:
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

### Sharing Env Vars Between Commands

Act going to run commands in independent shell environments to allow parallel execution as discussed in the previous section. That way if we need to share variables between commands (or acts) we can write variables as `key=val` strings to a special dotenv file which location is provided by `$ACT_ENV` var. Here an example:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    start:
      - grep -q MY_VAR $ACT_ENV || printf "MY_VAR=Bruno\n" >> $ACT_ENV
      - echo "MY_VAR is $MY_VAR"
```

The first command going to check if `MY_VAR` is already set in `ACT_ENV` and if not we going to add this variable and its value to `ACT_ENV`. This way when we run `act run foo` we should see `MY_VAR is nosebit` printed to the screen.

**NOTE**: Variables set to `ACT_ENV` going to be persisted as long the act is running. On stop/finish of act execution the main act process going to deleted all variables set. If we need to persist variables see the next section on loading variables from a specific file (not managed by act process itself).

**WARNING**: Remember to always add a line break char `\n` at the end of text you are appending to $ACT_ENV. Otherwise the variable not going to be loaded correctly.

**WARNING**: Be careful with race conditions when reading/write variables to `$ACT_ENV` when running commands in parallel.


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
    start:
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

With this env file set we can persist variables between different executions of an act. So, if we have the following:

```yaml
# actfile.yml
version: 1

envfile: .var

acts:
  foo:
    start:
      - grep -q MY_VAR $ACT_ENV_FILE || printf "MY_VAR=Bruno\n" >> $ACT_ENV_FILE
      - echo "MY_VAR is $MY_VAR"
```

then we can execute `act run foo` multiple times and `MY_VAR` going to be persisted. Note that we ca use `ACT_ENV_FILE` variable to reference our env file.


**WARNING**: Remember to always add a line break char `\n` at the end of text you are appending to $ACT_ENV_FILE. Otherwise the variable not going to be loaded correctly.


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
    start:
      - echo "daemon boolean flag => $FLAG_DAEMON"
      - echo "name string flag => $FLAG_NAME"
      - echo "other args are => $@"
```

This way we can run `act run foo -daemon -name=Bruno arg1 arg2`. Note that boolean flags which can be provided without values should have a default `false` added.


### Command Loops

If we need to run multiple commands that are very similar we can use loop functionality like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    start:
      - cmd: echo {{.LoopItem}}
        loop:
          items:
            - name1
            - name2
            - name3
```

The loop field also accepts a `glob` option we can use to set items to be the list of matched file paths like this:

```yaml
# actfile.yml
version: 1

acts:
  setup:
    start:
      - act: setup
        from: "{{.LoopItem}}"
        loop:
          glob: "**/actfile.yml"
        mismatch: allow
```

This way we going to loop over all subdiretories that has an `actfile.yml` in it and run the act named setup in those actfiles. Notice we used the `mismatch` field to prevent error in case actfile does not provide a `setup` rule.


### Log Mode

By default Act going to output logs in raw mode without any info about the act or timestamp. If we need prefix log output with act name and timestamp we can set `log` field to `prefixed` at act or actfile levels like this:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    log: prefixed
    start:
      - echo "im prefixed"
```

We can use the command line flag `l` as well to set log mode like this:

```bash
act run -l=prefixed test-unit
```


### Long Running Acts

If an act is written to be a long running process like the following:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    desc: This is a simple long running act example
    start:
      - while true; echo "Hello long running"; sleep 5; done
```

we can run it as a daemon using the following command:

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

Keep in mind that if we run `foo` act multiple times as daemons we going to endup having multiple running instances of the same act which is totally fine. But when running `act stop foo` we going to kill all `foo` instances at once. We can distinguish `foo` instances using `tags` flag like the following:

```bash
act run -d -t=foo-1 foo
act run -d -t=foo-2 foo
```

and then if we want to stop just `foo-1` instance we can use

```bash
act stop -t=foo-1 foo
```

which going to stop all instances of `foo` act which has tag `foo-1`.


### Detached Long Running Acts

If we need to run subacts as detached act processes which can be managed independently we can do like this:

```yaml
# actfile.yml
version: 1

acts:
  long1:
    start: while true; do echo "hello long1"; sleep 4; done
  long2:
    start: while true; do echo "hello long2"; sleep 2; done

  all:
    log: prefixed
    start:
      parallel: true
      cmds:
        - act: long1
          detach: true

        - act: long2
          detach: true
```

This way if we run `act run all` we going to run long1 and long2 as different act processes and we can stop only one of those with `act stop all::long1` for example. If we want to kill everything we can do `act stop all`.

### Teardown

If we need to run commands at the very end of the act execution we can use teardown (or final) commands like the following:

```yaml
# actfile.yml
version: 1

acts:
  foo:
    start: echo "started"
    final: echo "cleaning up"
```

Remember that teardown commands run if start command finish successfully or if it fails as well.
