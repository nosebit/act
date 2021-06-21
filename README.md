# Act

Act is a task runner and supervisor written in Go which aims to provide the following features:

* process supervision in a project level
* allow tasks to be written as simple bash scripting if wanted
* sub tasks which can be invoked like `act run cmd1 cmd2`
* dynamically include tasks defined in other locations allowing spliting task definition
* regex match task names

## Instalation

TBD

## How to Use

After installing `act` we create an `actfile.yml` file in root dir of our project so we can describe the acts we can run like the following:

```yaml
# actfile.yml
version: 1

acts:

  hello:
    desc: This is a simple hello world act example
    cmds:
      - echo "Hello act"
```

and then we run the act with the following command:

```bash
act run hello
```

If we don't specify `cmds` in `actfile.yml` file for `hello` then act going to look up for a shell script file named `hello.sh` in `acts` folder and if that script file is found then act going to run it which allows very powerful acts.

### Long Running Acts

If an act is written to be a long running process like the following:

```yaml
# actfile.yml
version: 1

acts:

  long:
    desc: This is a simple long running act example
    cmds:
      - while true; echo "Hello long running"; sleep 5; done
```

then we can run it as a daemon using the following command:

```bash
act run -d long
```

To stop the act is as simple as running

```bash
act stop long
```