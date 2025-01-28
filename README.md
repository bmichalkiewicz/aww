# aww CLI Tool

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
  skip: <true|false> # action
  commit: <string> # action
  push: <true|false> # action
  projects:
    - url: <project_name_1>
      commit: <string> # action
      push: <true|false> # action
    - url: <project_name_2>
      commit: <string> # action
      push: <true|false> # action
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
