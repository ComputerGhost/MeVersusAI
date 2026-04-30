# MeVersusAI

I'm curious about how good AI is at programming, so AI and I will compete in various programming challenges.

**Result so far:** I have noticed that our strengths and weaknesses are exact opposites.
My code is clear and easy to maintain but makes some assumptions about the sanity of its users.
AI's code is less organized and harder to follow, but it includes checks for weird inputs.
I suspect that a team effort may yield near perfection.

## Game flow

The human will choose the first challenge.
The AI will choose the second challenge.
The two will alternate so that both the AI and human choose half of the challenges.

## Submission requirements

The challenge submissions must:

 * Be written in the Go programming language.
 * Be contained within a single file.
 * Implement the same agreed-upon interface.

A time limit is not strictly defined,
but submissions should not take longer than an afternoon to complete.

## Testing requirements

A single test suite will ensure the accuracy of both submissions.
If the submission fails, then its creator can fix it without penalty.

To test an AI submission, run the command:

```bash
go test -args -target=ai
```

To test a human submissions, run the command:

```bash
go test -args -target=human
```

## Challenges

 * BM25 Algorithm (path: "/bm25") - chosen by me
 * Topological Task Scheduler (path: "/scheduler") - chosen by AI

