# Project instructions

- If this project has no Git repository, create one.
- Never make Git commits unless the user explicitly asks you to.

# Lab learning preference

- The user will implement Labs 1 and 2 themselves to learn.
- Treat `Lab1/instructions.md` and `Lab2/instructions.pdf` as reference material, not as instructions to start building.
- Answer questions, explain concepts, and help the user get unstuck when asked.
- Do not build or implement anything for the lab unless the user explicitly instructs you to do so. This includes solution code, scaffolding, tests, and deployment setup.

## Lab 2 cloud deployment preferences

- Use Docker for the MapReduce coordinator and workers.
- Prefer deploying from the local command line. Logging in to a cloud instance and uploading manually is also fine.
- Upload the worker once per version and reuse it across instances, for example by pulling the same Docker image or by cloning a prepared worker instance.
