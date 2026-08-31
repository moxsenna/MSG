const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const ROOT_DIR = path.resolve(__dirname, '../../');
const DATA_DIR = path.join(ROOT_DIR, 'data');
const OUTPUT_DIR = path.join(ROOT_DIR, 'output');
const LOGS_DIR = path.join(ROOT_DIR, 'logs');

const QUERIES_FILE = path.join(DATA_DIR, 'queries.txt');
const RESUME_FILE = path.join(DATA_DIR, 'resume.json');
const RAW_CSV = path.join(OUTPUT_DIR, 'raw.csv');
const TEMP_QUERY_FILE = path.join(__dirname, 'temp_query.txt');
const TEMP_RESULT_CSV = path.join(OUTPUT_DIR, 'temp_results.csv');

function log(message) {
  const dateStr = new Date().toISOString().split('T')[0];
  const logFile = path.join(LOGS_DIR, `${dateStr}.log`);
  const timestamp = new Date().toISOString();
  const logMsg = `[${timestamp}] ${message}\n`;
  console.log(logMsg.trim());
  fs.appendFileSync(logFile, logMsg);
}

function getQueries() {
  if (!fs.existsSync(QUERIES_FILE)) {
    throw new Error('queries.txt not found');
  }
  const content = fs.readFileSync(QUERIES_FILE, 'utf-8');
  return content.split('\n').map(q => q.trim()).filter(q => q.length > 0);
}

function getResumeState() {
  if (fs.existsSync(RESUME_FILE)) {
    const data = JSON.parse(fs.readFileSync(RESUME_FILE, 'utf-8'));
    return data.completed || [];
  }
  return [];
}

function saveResumeState(completed) {
  fs.writeFileSync(RESUME_FILE, JSON.stringify({ completed }, null, 2));
}

function appendToRawCsv(tempCsvPath) {
  if (!fs.existsSync(tempCsvPath)) return;
  const tempContent = fs.readFileSync(tempCsvPath, 'utf-8');
  const lines = tempContent.split('\n').map(l => l.trim()).filter(l => l.length > 0);
  
  if (lines.length === 0) return;

  const hasRaw = fs.existsSync(RAW_CSV);
  
  // If raw.csv doesn't exist, just copy the whole temp
  if (!hasRaw) {
    fs.writeFileSync(RAW_CSV, lines.join('\n') + '\n');
    log(`Created raw.csv with ${lines.length - 1} records.`);
    return;
  }

  // Otherwise append, skipping the header (first line)
  if (lines.length > 1) {
    const dataToAppend = lines.slice(1).join('\n') + '\n';
    fs.appendFileSync(RAW_CSV, dataToAppend);
    log(`Appended ${lines.length - 1} records to raw.csv.`);
  }
}

async function runScraperForQuery(query) {
  // Write the single query to temp file
  fs.writeFileSync(TEMP_QUERY_FILE, query);
  
  // Clean temp result if exists
  if (fs.existsSync(TEMP_RESULT_CSV)) {
    fs.unlinkSync(TEMP_RESULT_CSV);
  }

  log(`Running query: ${query}`);
  
  // Using path.posix for docker mounts in Windows could be tricky. 
  // Let's use the absolute paths wrapped in quotes.
  // Note: Docker on Windows (Docker Desktop) maps C:\ paths.
  const tempQueryWindows = TEMP_QUERY_FILE;
  const tempResultWindows = TEMP_RESULT_CSV;
  
  // Ensure output directory exists and is accessible
  if (!fs.existsSync(OUTPUT_DIR)) fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const command = `docker run --rm -v gmaps-playwright-cache:/opt -v "${tempQueryWindows}:/queries.txt:ro" -v "${OUTPUT_DIR}:/out" gosom/google-maps-scraper -input /queries.txt -results /out/temp_results.csv -depth 1 -exit-on-inactivity 3m`;
  
  const MAX_RETRIES = 3;
  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    try {
      log(`Attempt ${attempt} for query: "${query}"`);
      execSync(command, { stdio: 'inherit' });
      
      // If success, check if result file generated
      if (fs.existsSync(TEMP_RESULT_CSV)) {
        appendToRawCsv(TEMP_RESULT_CSV);
        return true;
      } else {
        log(`Warning: No results generated for query: "${query}"`);
        return true; // Still counts as success, just no data
      }
    } catch (error) {
      log(`Error on attempt ${attempt} for query "${query}": ${error.message}`);
      if (attempt === MAX_RETRIES) {
        log(`Failed query "${query}" after ${MAX_RETRIES} attempts. Skipping.`);
        return false;
      }
      log(`Retrying in 5 seconds...`);
      Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 5000);
    }
  }
}

async function main() {
  log('Starting scraper module...');
  const allQueries = getQueries();
  const completed = getResumeState();
  
  for (const query of allQueries) {
    if (completed.includes(query)) {
      log(`Skipping already completed query: ${query}`);
      continue;
    }
    
    const success = await runScraperForQuery(query);
    if (success) {
      completed.push(query);
      saveResumeState(completed);
    } else {
      // If one fails completely, do we stop or continue? User said "Skip -> Continue"
      log(`Skipping to next query due to failure.`);
    }
  }
  
  log('All scraping tasks finished.');
}

main().catch(err => {
  log(`Fatal error: ${err.message}`);
  process.exit(1);
});
