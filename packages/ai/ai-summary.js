const { GoogleGenerativeAI } = require('@google/generative-ai');

/**
 * Generate a short summary and opportunity pitch for a business lead using Gemini AI.
 * @param {Object} leadData - The combined data (raw maps data + audit data + score)
 * @returns {Promise<string>} The AI summary text
 */
async function generateSummary(leadData) {
  const apiKey = process.env.GEMINI_API_KEY;
  if (!apiKey) {
    return 'Error: GEMINI_API_KEY is not set in environment.';
  }

  const genAI = new GoogleGenerativeAI(apiKey);
  // For standard text tasks, gemini-1.5-flash is fast and cost-effective
  const model = genAI.getGenerativeModel({ model: 'gemini-1.5-flash' });

  const prompt = `
Anda adalah seorang analis bisnis dan konsultan digital B2B.
Tugas Anda adalah membuat 1 kalimat ringkasan (maksimal 25 kata) mengenai prospek bisnis berikut. Fokus pada peluang layanan pembuatan website atau optimasi digital.

Data Prospek:
- Nama: ${leadData.name}
- Kategori: ${leadData.category}
- Rating Maps: ${leadData.rating}
- Jumlah Ulasan Maps: ${leadData.reviews}
- Punya Website: ${leadData.hasWebsite ? 'Ya' : 'Tidak'}

Contoh output jika tidak punya website:
"Website belum tersedia, reputasi Google Maps kuat dengan ${leadData.reviews} ulasan. Peluang besar untuk menawarkan website company profile."

Contoh output jika punya website tapi buruk:
"Website sudah ada namun kurang optimal. Peluang untuk menawarkan redesign website."

Buat ringkasan untuk prospek di atas:
  `;

  try {
    const result = await model.generateContent(prompt);
    const response = await result.response;
    return response.text().trim();
  } catch (err) {
    console.error('Error generating AI summary:', err);
    return 'Gagal memproses AI summary.';
  }
}

module.exports = {
  generateSummary
};
