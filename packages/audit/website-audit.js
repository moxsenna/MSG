const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

const SCREENSHOTS_DIR = path.resolve(__dirname, '../../screenshots');

/**
 * Audit a website.
 * @param {string} businessName - Name of the business for screenshot filename
 * @param {string} url - Website URL
 * @returns {Promise<Object>} Audit results
 */
async function auditWebsite(businessName, url) {
  if (!url) {
    return { hasWebsite: false };
  }

  // Ensure url has http/https
  let targetUrl = url;
  if (!targetUrl.startsWith('http')) {
    targetUrl = 'http://' + targetUrl;
  }

  const result = {
    hasWebsite: true,
    originalUrl: url,
    isHttps: targetUrl.startsWith('https') || false,
    title: null,
    hasMetaDescription: false,
    hasContactInfo: false,
    hasSocialLinks: false,
    isResponsive: true, // simplified assumption unless tested with specific viewport
    screenshotPath: null,
    error: null
  };

  let browser;
  try {
    browser = await chromium.launch({ headless: true });
    const context = await browser.newContext({
      viewport: { width: 375, height: 812 }, // Mobile viewport to check responsiveness
      userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1'
    });
    
    const page = await context.newPage();
    
    // Set a timeout of 15 seconds to avoid hanging on slow sites
    await page.goto(targetUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
    
    // Check HTTPS if it redirected
    const finalUrl = page.url();
    if (finalUrl.startsWith('https')) {
      result.isHttps = true;
    }

    // Get Title
    result.title = await page.title();

    // Check Meta Description
    const metaDesc = await page.$("meta[name='description']");
    result.hasMetaDescription = !!metaDesc;

    // Check Contact Info (simple regex on body text)
    const bodyText = await page.innerText('body');
    const emailRegex = /[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}/;
    const phoneRegex = /(?:\+62|62|0)[2-9][0-9]{7,11}/; // basic indonesian phone regex
    
    result.hasContactInfo = emailRegex.test(bodyText) || phoneRegex.test(bodyText);

    // Take screenshot
    if (!fs.existsSync(SCREENSHOTS_DIR)) {
      fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });
    }
    const safeName = businessName.replace(/[^a-z0-9]/gi, '_').toLowerCase();
    const screenshotFileName = `${safeName}_mobile.png`;
    const screenshotPath = path.join(SCREENSHOTS_DIR, screenshotFileName);
    
    await page.screenshot({ path: screenshotPath, fullPage: false });
    result.screenshotPath = screenshotPath;

  } catch (err) {
    result.error = err.message;
  } finally {
    if (browser) {
      await browser.close();
    }
  }

  return result;
}

module.exports = {
  auditWebsite
};
