/**
 * app.js — Online Code Execution System Frontend
 *
 * How it works:
 * 1. User selects a language and enters code.
 * 2. On "Run": POST /api/jobs  → receive job ID
 * 3. Poll GET /api/jobs/{id} every 500ms until status is terminal.
 * 4. Display stdout, stderr, exit code, execution time.
 *
 * Communication: uses fetch() (native browser API, no libraries needed).
 */

// ============================================================
// Constants
// ============================================================

const API_BASE = '';   // same origin
const POLL_INTERVAL_MS = 500;   // how often we check job status

// Terminal statuses — polling stops when we reach one of these.
const TERMINAL_STATUSES = new Set([
  'COMPLETED', 'COMPILE_ERROR', 'RUNTIME_ERROR', 'TIMEOUT', 'FAILED'
]);

// ============================================================
// Code examples for each language
// ============================================================

const EXAMPLES = {
  python: `# Python example — reads two numbers and prints their sum
import sys

a = int(input("Enter a: "))
b = int(input("Enter b: "))
print(f"Sum: {a + b}")
`,

  go: `// Go example — Hello World with goroutine
package main

import (
\t"fmt"
\t"sync"
)

func main() {
\tvar wg sync.WaitGroup
\tfor i := 1; i <= 3; i++ {
\t\twg.Add(1)
\t\tgo func(n int) {
\t\t\tdefer wg.Done()
\t\t\tfmt.Printf("Hello from goroutine %d\\n", n)
\t\t}(i)
\t}
\twg.Wait()
\tfmt.Println("All goroutines done!")
}
`,

  cpp: `// C++ example — compute Fibonacci
#include <iostream>
using namespace std;

int fib(int n) {
    if (n <= 1) return n;
    return fib(n-1) + fib(n-2);
}

int main() {
    for (int i = 0; i <= 10; i++) {
        cout << "fib(" << i << ") = " << fib(i) << endl;
    }
    return 0;
}
`,

  java: `// Java example — Hello World
public class Main {
    public static void main(String[] args) {
        System.out.println("Hello from Java!");
        for (int i = 1; i <= 5; i++) {
            System.out.println("Line " + i);
        }
    }
}
`,
};

// ============================================================
// State
// ============================================================

let currentPollTimer = null;
let pollingVersion = 0;

// ============================================================
// DOM helpers
// ============================================================

const $ = id => document.getElementById(id);

function setStatus(status) {
  const badge = $('statusBadge');
  const text  = $('statusText');

  // Remove all status- classes
  badge.className = 'status-badge';
  badge.classList.add('status-' + status.toLowerCase());
  text.textContent = status.replace(/_/g, ' ');
}

function showProgress(label, percent) {
  $('progressSection').classList.remove('hidden');
  $('progressLabel').textContent = label;
  $('progressFill').style.width = percent + '%';
}

function hideProgress() {
  $('progressSection').classList.add('hidden');
}

function showJobInfo(job) {
  $('jobInfo').classList.remove('hidden');
  $('jobId').textContent = job.id.substring(0, 13) + '…';
  $('jobId').title = job.id;
  $('jobLang').textContent = job.language;
  $('exitCode').textContent = job.exit_code ?? '—';
  $('execTime').textContent = job.execution_time_ms ? job.execution_time_ms + 'ms' : '—';
}

function setRunning(loading) {
  const btn  = $('runBtn');
  const text = $('runBtnText');
  btn.disabled = loading;
  text.textContent = loading ? 'Running…' : 'Run Code';
}

function showTab(name) {
  ['stdout', 'stderr', 'error'].forEach(tab => {
    $('pane' + cap(tab)).classList.toggle('hidden', tab !== name);
    $('tab' + cap(tab)).classList.toggle('active', tab === name);
  });
}

function cap(s) { return s.charAt(0).toUpperCase() + s.slice(1); }

// ============================================================
// Line numbers (sync with code editor scroll)
// ============================================================

const codeEditor = $('codeEditor');
const lineNumbers = $('lineNumbers');

function updateLineNumbers() {
  const lines = codeEditor.value.split('\n').length;
  lineNumbers.textContent = Array.from({ length: lines }, (_, i) => i + 1).join('\n');
}

codeEditor.addEventListener('input', updateLineNumbers);
codeEditor.addEventListener('scroll', () => {
  lineNumbers.scrollTop = codeEditor.scrollTop;
});

// Tab key inserts spaces (not browser's default focus change)
codeEditor.addEventListener('keydown', e => {
  if (e.key === 'Tab') {
    e.preventDefault();
    const start = codeEditor.selectionStart;
    const end   = codeEditor.selectionEnd;
    codeEditor.value = codeEditor.value.substring(0, start) + '  ' + codeEditor.value.substring(end);
    codeEditor.selectionStart = codeEditor.selectionEnd = start + 2;
    updateLineNumbers();
  }
});

// ============================================================
// Example loader
// ============================================================

function loadExample() {
  const lang = $('languageSelect').value;
  codeEditor.value = EXAMPLES[lang] || '';
  // Set corresponding stdin example
  if (lang === 'python') {
    $('stdinInput').value = '5\n3';
  } else {
    $('stdinInput').value = '';
  }
  updateLineNumbers();
}

// Populate example on language change
$('languageSelect').addEventListener('change', () => {
  if (codeEditor.value.trim() === '' || Object.values(EXAMPLES).includes(codeEditor.value)) {
    loadExample();
  }
});

// Load a default example on page load
window.addEventListener('DOMContentLoaded', () => {
  loadExample();
  pollHealth();
  setInterval(pollHealth, 5000); // refresh health every 5 seconds
});

// ============================================================
// Clear
// ============================================================

function clearAll() {
  codeEditor.value = '';
  $('stdinInput').value = '';
  $('stdoutContent').innerHTML = '<span class="placeholder-text">Run your code to see output here...</span>';
  $('stderrContent').innerHTML = '<span class="placeholder-text">Errors will appear here...</span>';
  $('errorContent').innerHTML  = '<span class="placeholder-text">Error details will appear here...</span>';
  $('jobInfo').classList.add('hidden');
  hideProgress();
  setStatus('idle');
  setRunning(false);
  updateLineNumbers();
  stopPolling();
}

// ============================================================
// Copy output
// ============================================================

function copyOutput(which) {
  const el = which === 'stdout' ? $('stdoutContent') : $('stderrContent');
  navigator.clipboard.writeText(el.textContent).then(() => {
    const btn = el.parentElement.querySelector('.copy-btn');
    const orig = btn.textContent;
    btn.textContent = 'Copied!';
    setTimeout(() => btn.textContent = orig, 1500);
  });
}

// ============================================================
// Submit Job
// ============================================================

async function submitJob() {
  const language = $('languageSelect').value;
  const code     = codeEditor.value.trim();
  const stdin    = $('stdinInput').value;

  if (!code) {
    alert('Please enter some code first!');
    return;
  }

  // Reset UI
  stopPolling();
  const version = pollingVersion;
  setRunning(true);
  setStatus('QUEUED');
  showProgress('Submitting job to server…', 10);
  showTab('stdout');

  $('stdoutContent').innerHTML = '<span class="placeholder-text">Waiting for result...</span>';
  $('stderrContent').innerHTML = '<span class="placeholder-text">Waiting for result...</span>';
  $('errorContent').innerHTML  = '<span class="placeholder-text">Waiting for result...</span>';

  try {
    // POST /api/jobs
    const res = await fetch(`${API_BASE}/api/jobs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ language, code, stdin }),
    });

    const data = await res.json();
    if (version !== pollingVersion) return;

    if (!res.ok) {
      throw new Error(data.error || `Server error: ${res.status}`);
    }

    const jobId = data.id;
    showProgress('Job queued — waiting for a worker…', 30);
    $('jobInfo').classList.remove('hidden');
    $('jobId').textContent = jobId.substring(0, 13) + '…';
    $('jobId').title = jobId;
    $('jobLang').textContent = language;

    // Start polling for result
    startPolling(jobId, version);

  } catch (err) {
    if (version === pollingVersion) {
      handleError(err.message || 'Network error — is the server running?');
    }
  }
}

// ============================================================
// Polling
// ============================================================

function startPolling(jobId, version) {
  currentPollTimer = setTimeout(() => pollJob(jobId, version), POLL_INTERVAL_MS);
}

function stopPolling() {
  if (currentPollTimer) {
    clearTimeout(currentPollTimer);
    currentPollTimer = null;
  }
  pollingVersion++;
}

async function pollJob(jobId, version) {
  if (version !== pollingVersion) return;

  try {
    const res  = await fetch(`${API_BASE}/api/jobs/${jobId}`);
    const job  = await res.json();
    if (version !== pollingVersion) return;

    if (!res.ok) {
      throw new Error(job.error || `Lookup failed: ${res.status}`);
    }

    // Update progress bar based on status
    const progressMap = {
      'QUEUED':        { label: 'Job queued — waiting for a worker…', pct: 30 },
      'RUNNING':       { label: 'Worker is executing your code…',      pct: 65 },
    };
    const prog = progressMap[job.status];
    if (prog) showProgress(prog.label, prog.pct);

    setStatus(job.status);
    showJobInfo(job);

    if (TERMINAL_STATUSES.has(job.status)) {
      // Job is done — display results
      hideProgress();
      setRunning(false);
      displayResult(job);
      return; // stop polling
    }

    // Not done yet — poll again
    currentPollTimer = setTimeout(() => pollJob(jobId, version), POLL_INTERVAL_MS);

  } catch (err) {
    if (version === pollingVersion) {
      handleError(err.message);
    }
  }
}

// ============================================================
// Display Result
// ============================================================

function displayResult(job) {
  // stdout
  const stdout = job.stdout || '';
  $('stdoutContent').textContent = stdout || '(no output)';

  // stderr
  const stderr = job.stderr || '';
  $('stderrContent').textContent = stderr || '(no output)';
  if (stderr) $('tabStderr').classList.add('has-content');

  // error info
  const errorLines = [];
  if (job.status !== 'COMPLETED') {
    errorLines.push(`Status: ${job.status}`);
    if (job.error_message) errorLines.push(`Error: ${job.error_message}`);
    errorLines.push(`Exit Code: ${job.exit_code}`);
    errorLines.push(`Execution Time: ${job.execution_time_ms}ms`);
    if (job.status === 'COMPILE_ERROR') {
      errorLines.push('');
      errorLines.push('The code could not be compiled. Check stderr for compiler output.');
    } else if (job.status === 'RUNTIME_ERROR') {
      errorLines.push('');
      errorLines.push('The code compiled but crashed at runtime. Check stderr for the error.');
    } else if (job.status === 'TIMEOUT') {
      errorLines.push('');
      errorLines.push('The code exceeded the maximum execution time (10 seconds).');
      errorLines.push('Check for infinite loops or excessive computation.');
    }
  } else {
    errorLines.push('Status: COMPLETED (success)');
    errorLines.push(`Exit Code: ${job.exit_code}`);
    errorLines.push(`Execution Time: ${job.execution_time_ms}ms`);
  }
  $('errorContent').textContent = errorLines.join('\n');

  // Auto-switch to the most relevant tab
  if (job.status === 'COMPILE_ERROR' || job.status === 'RUNTIME_ERROR') {
    showTab('stderr');
  } else if (job.status === 'TIMEOUT' || job.status === 'FAILED') {
    showTab('error');
  } else {
    showTab('stdout');
  }
}

// ============================================================
// Error handling
// ============================================================

function handleError(message) {
  setRunning(false);
  hideProgress();
  setStatus('FAILED');
  $('errorContent').textContent = message;
  showTab('error');
  console.error('OCES error:', message);
}

// ============================================================
// Health polling (status bar)
// ============================================================

async function pollHealth() {
  try {
    const res  = await fetch(`${API_BASE}/api/health`);
    const data = await res.json();

    $('queueLen').textContent  = data.queue_len  ?? '?';
    $('queueCap').textContent  = data.queue_cap  ?? '?';
    $('totalJobs').textContent = data.jobs_total ?? '?';

    $('serverStatusDot').className  = 'server-dot dot-ok';
    $('serverStatusText').textContent = 'Server online';
  } catch {
    $('serverStatusDot').className  = 'server-dot dot-error';
    $('serverStatusText').textContent = 'Server offline';
  }
}
