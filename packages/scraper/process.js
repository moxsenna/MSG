const fs = require('fs');
const path = require('path');
const csv = require('csv-parser');
const createCsvWriter = require('csv-writer').createObjectCsvWriter;
const { auditWebsite } = require('../audit/website-audit');
const { calculateScore } = require('../scoring/lead-score');
const { generateSummary } = require('../ai/ai-summary');

const ROOT_DIR = path.resolve(__dirname, '../../');
const OUTPUT_DIR = path.join(ROOT_DIR, 'output');
const RAW_CSV = path.join(OUTPUT_DIR, 'raw.csv');
const FINAL_CSV = path.join(OUTPUT_DIR, 'leads.csv');
const FINAL_JSON = path.join(OUTPUT_DIR, 'leads.json');
const STATE_FILE = path.join(OUTPUT_DIR, 'process_state.json');

async function processLeads() {
  console.log('Starting Lead Processing Pipeline...');

  if (!fs.existsSync(RAW_CSV)) {
    console.error('raw.csv not found. Please run the scraper first.');
    process.exit(1);
  }

  // Load state to allow resuming the processing phase
  let processedIds = new Set();
  let finalResults = [];

  if (fs.existsSync(STATE_FILE)) {
    const state = JSON.parse(fs.readFileSync(STATE_FILE, 'utf-8'));
    processedIds = new Set(state.processedIds || []);
  }

  if (fs.existsSync(FINAL_JSON)) {
    finalResults = JSON.parse(fs.readFileSync(FINAL_JSON, 'utf-8'));
  }

  const rawData = [];
  
  // Read all rows
  await new Promise((resolve, reject) => {
    fs.createReadStream(RAW_CSV)
      .pipe(csv())
      .on('data', (data) => rawData.push(data))
      .on('end', () => resolve())
      .on('error', (err) => reject(err));
  });

  console.log(`Found ${rawData.length} leads in raw.csv. Starting pipeline...`);

  // Process each lead sequentially (can be optimized to parallel later)
  for (let i = 0; i < rawData.length; i++) {
    const lead = rawData[i];
    
    // We need a unique ID, let's use Maps URL or a combination
    const id = lead.place_id || lead.title || `row_${i}`;
    
    if (processedIds.has(id)) {
      continue;
    }

    console.log(`Processing [${i+1}/${rawData.length}]: ${lead.title}`);

    // 1. Audit
    console.log(` -> Auditing website: ${lead.website || 'N/A'}`);
    const auditResult = await auditWebsite(lead.title, lead.website);

    // 2. Score
    console.log(` -> Calculating score...`);
    // Normalize some fields for scoring
    const normalizedData = {
      name: lead.title,
      category: lead.category,
      rating: lead.rating,
      reviews: lead.reviews,
      phone: lead.phone,
      status: lead.status
    };
    
    const { score, priority } = calculateScore(normalizedData, auditResult);

    // 3. AI Summary
    console.log(` -> Generating AI summary...`);
    const aiSummary = await generateSummary({
      ...normalizedData,
      hasWebsite: auditResult.hasWebsite
    });

    // 4. Combine Result
    const finalLead = {
      ...normalizedData,
      website: lead.website,
      maps_url: lead.url,
      address: lead.address,
      
      audit_https: auditResult.isHttps,
      audit_mobile: auditResult.isResponsive,
      audit_contact: auditResult.hasContactInfo,
      screenshot: auditResult.screenshotPath,
      
      score,
      priority,
      ai_summary: aiSummary
    };

    finalResults.push(finalLead);
    processedIds.add(id);

    // Save incrementally
    fs.writeFileSync(FINAL_JSON, JSON.stringify(finalResults, null, 2));
    fs.writeFileSync(STATE_FILE, JSON.stringify({ processedIds: Array.from(processedIds) }, null, 2));

    // Save to CSV
    const csvWriter = createCsvWriter({
      path: FINAL_CSV,
      header: [
        { id: 'name', title: 'Business Name' },
        { id: 'category', title: 'Category' },
        { id: 'phone', title: 'Phone' },
        { id: 'address', title: 'Address' },
        { id: 'website', title: 'Website' },
        { id: 'maps_url', title: 'Google Maps URL' },
        { id: 'rating', title: 'Rating' },
        { id: 'reviews', title: 'Reviews' },
        { id: 'score', title: 'Score' },
        { id: 'priority', title: 'Priority' },
        { id: 'ai_summary', title: 'AI Summary' }
      ]
    });
    await csvWriter.writeRecords(finalResults);

    console.log(` -> Done. Score: ${score}, Priority: ${priority}`);
  }

  console.log('Pipeline finished successfully!');
}

processLeads().catch(err => console.error('Pipeline error:', err));
