# aww

![aww Logo](assets/logo.png)

## Introduction

`aww` is a tool (something like multi-git) to interact with repositories declared in the file (created by [brr](https://github.com/bmichalkiewicz/brr))

## Features

- Clone repositories from the repositories file.
- Find repositories that meet a given condition (unpushed, uncommitted, empty)
- Switch branches specified by user (or default branch for specific branching strategy)
- Do actions specified in the file repository file

## Git repositories

```yaml
- name: <group_name>
  actions:
    skip: <true|false>
    commit: <string>
    push: <true|false>
  projects:
    - url: <project_name_1>
      actions:
        skip: <true|false>
        commit: <string>
        push: <true|false>
    - url: <project_name_2>
      actions:
        skip: <true|false>
        commit: <string>
        push: <true|false>
```

if you want to clean repositories file after doing actions, just do
```bash
aww actions reset
```

## Commands

```bash
aww --help
aww git --help
```
