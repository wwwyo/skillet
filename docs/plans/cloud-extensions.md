# Skillet Cloud Extensions Vision

## Why (Why Separated Experiences Are Needed)

### 1. The Skill Challenge Is "Persona Difference," Not "Sharing"
- Skill creation, management, and sharing already work for engineers
- The real issue is not there—it is that
  - **Personas who work with Skills are fundamentally different**

### 2. Engineers and Non-Engineers See Skills Differently

| Aspect | Engineer | Non-Engineer |
|--------|----------|--------------|
| How they view Skills | Structure, config, composition units | Means to make work easier |
| Primary concern | Reproducibility, extensibility, management | Whether it works, whether it's safe |
| Action | Files / Git / CLI | GUI / Forms |
| Experimentation | Fast, self-responsible | High cost of failure |

- Facing the same Skill,
  - Engineers look at "how it's built"
  - Non-engineers look at "whether it helps my current work"

### 3. Engineers Won't Be the Only Ones Using Skills Going Forward
- As GUIs like Claude Code / cowork become common,
  - Non-engineers will also become those who "operate" AI
- At that point,
  - Making them use the same Skill representation as engineers feels unnatural
- Skills themselves can be shared,
  - But **the entry points (how to create, interact, share) must be separated**

### 4. One Experience Cannot Satisfy Everyone
- Experiences optimized for engineers
  - Are overwhelming and difficult for non-engineers
- Over-simplifying for non-engineers
  - Makes them too restrictive for engineers
- Therefore,
  - **Skills are shared**
  - **Experiences are separated**
  This design is necessary

## What (What We Provide)

### 1. Shared Skill Foundation (Engineer-Oriented OSS)
- OSS to define, compose, and manage Skills in a structured way
- Assumes Git / CLI / file-based workflows
- Ensures reproducibility, portability, extensibility
- Foundation for engineers to design and maintain Skills

### 2. Cloud Experience for Non-Engineers
- Same Skills, but
  - Business-focused
  - GUI
  - Safe operations
  Provided via a Cloud UI
- Non-engineers use Skills not as
  - Configuration or structure
  - But as **business templates and business buttons**

### 3. One Skill, Multiple Entry Points
- Skills themselves share a common spec and representation
- However,
  - How to create
  - How to use
  - How to share and improve
  are optimized per persona
- So engineers and non-engineers can
  - Work with the same Skill
  - At different layers

### 4. Target State
- Engineers
  - Design Skills
  - Stabilize them
  - Extend them
- Non-engineers
  - Apply Skills to work
  - Improve them
  - Roll them out to the team

> Skills are shared assets.
> But usage and entry points
> **are separated by persona**
