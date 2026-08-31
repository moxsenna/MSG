/**
 * Calculate lead score based on raw data and audit results.
 * @param {Object} rawData - Data from google maps (rating, reviews, phone, etc)
 * @param {Object} auditResult - Data from website audit
 * @returns {Object} { score, priority }
 */
function calculateScore(rawData, auditResult) {
  let score = 0;

  // 1. Website absence is a huge opportunity
  if (!auditResult.hasWebsite) {
    score += 50;
  } else {
    // If they have a website but it's not optimal
    if (!auditResult.isHttps) score += 10;
    if (!auditResult.hasMetaDescription) score += 5;
    if (!auditResult.hasContactInfo) score += 5;
  }

  // 2. Google Maps data
  const rating = parseFloat(rawData.rating) || 0;
  const reviews = parseInt(rawData.reviews) || 0;

  if (rating > 4.5) {
    score += 15;
  } else if (rating > 4.0) {
    score += 5;
  }

  if (reviews > 100) {
    score += 15;
  } else if (reviews > 50) {
    score += 5;
  }

  // 3. Contact info (WhatsApp potential)
  // Assuming Indonesian mobile numbers (08 or +628) indicate WA
  const phone = rawData.phone || '';
  const isMobile = /(?:\+628|628|08)[0-9]{7,11}/.test(phone.replace(/\D/g, ''));
  if (isMobile) {
    score += 10;
  }

  // 4. Active business 
  // If we found it on maps and it's not explicitly closed, we assume active
  // gosom scraper returns "status" or similar if closed. If not closed, +10.
  const isClosed = (rawData.status || '').toLowerCase().includes('closed');
  if (!isClosed) {
    score += 10;
  }

  // Priority threshold
  let priority = 'Low';
  if (score >= 80) {
    priority = 'Hot Lead';
  } else if (score >= 50) {
    priority = 'Warm Lead';
  }

  return {
    score,
    priority
  };
}

module.exports = {
  calculateScore
};
