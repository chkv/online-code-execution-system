# Presentation Script

**[Slide 1: Title]**
"Hello everyone, I'm the lead engineer for the Online Code Execution System. Today I'm going to show you a platform we built that allows users to submit code in multiple languages, and executes it securely in the cloud."

**[Slide 2: The Problem]**
"Building a code execution engine is hard for two reasons. First, security: users can submit malicious code like infinite loops or scripts that try to delete system files. Second, concurrency: if 50 students submit code at the exact same time, the system needs to process them without locking up."

**[Slide 3: Live Demo]**
"Let's look at the system in action. *(Open the web app)*. Here is our frontend. I'll write a quick Python script to print a message. When I click Run, the frontend sends it to our Go backend, which queues it, executes it, and streams the result back. 
Now, let's look at concurrency. I'm going to run a script that submits 5 jobs simultaneously. Notice in the logs how our worker pool processes them in parallel."

**[Slide 4: Architecture - Concurrency]**
"How did we build this? The entire backend is written in Go.
When a request comes in, we put it into a Go 'Channel'. This acts as our Job Queue.
We then have a Worker Pool—a set of goroutines running in parallel. They sit in a loop, pulling jobs from the channel.
Because multiple workers are updating job statuses while the API is reading them, we had to prevent data races. We did this using a `sync.RWMutex`, which we rigorously tested using Go's built-in Race Detector."

**[Slide 5: Architecture - Security]**
"To solve the security problem, we don't run the code directly on the server. Instead, our Go backend talks directly to the Docker daemon. For every single job, it spins up a brand new, isolated Docker container.
We strip this container of all network access, drop its Linux capabilities so it can't escalate privileges, and hard-limit the CPU and RAM. After 10 seconds, if the code is still running—like an infinite loop—we kill the container automatically."

**[Slide 6: Inspirations]**
"We were inspired by open-source systems like 'Judex', which use Redis and microservices. However, we intentionally simplified the design into a single binary using Go's native channels and mutexes. This makes the system much easier to deploy and perfectly demonstrates the core concurrency concepts we learned in this course."

**[Slide 7: Conclusion]**
"In conclusion, we successfully built a secure, highly concurrent code execution system from scratch. Thank you, and I'd be happy to answer any questions."
