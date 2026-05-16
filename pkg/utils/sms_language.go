package utils

import (
	"strings"
)

// GetLocalizedSMS returns the SMS content in the patient's preferred language based on their home region.
func GetLocalizedSMS(templateKey string, homeRegion string, placeholders map[string]string) string {
	// Determine language based on home_region
	lang := "en"
	regionUpper := strings.ToUpper(homeRegion)
	if regionUpper == "AMHARA" {
		lang = "am"
	} else if regionUpper == "OROMIA" {
		lang = "om"
	}

	var templates map[string]string

	switch lang {
	case "am":
		templates = map[string]string{
			"accepted":    "ወደ {{Hospital}} – {{Department}} የተላከው ሪፈራል ተቀባይነት አግኝቷል። ለቀጠሮ ተቀባዩ ቡድን ያነጋግርዎታል። መለያ: {{ReferralID}}",
			"scheduled":   "ቀጠሮዎ በ {{Date}} በ {{Hospital}}፣ {{Department}} ነው። እባክዎ በሰዓቱ ይድረሱ። መለያ: {{ReferralID}}",
			"missed":      "በ {{Date}} በ {{Hospital}} የነበረዎትን ቀጠሮ አሳልፈዋል። እባክዎ በተቻለ ፍጥነት ሆስፒታሉን ይጎብኙ። ሪፈራልዎ እንደገና እየታየ ነው። መለያ: {{ReferralID}}",
			"rescheduled": "ቀጠሮዎ ወደ {{NewDate}} በ {{Hospital}}፣ {{Department}} ተቀይሯል። የቀድሞው ቀጠሮ ተሰርዟል። መለያ: {{ReferralID}}",
			"reminder":    "ማሳሰቢያ: ነገ በ {{Time}} በ {{Hospital}} ቀጠሮ አልዎት። እባክዎ መታወቂያዎን ይዘው ይምጡ። መለያ: {{ReferralID}}",
		}
	case "om":
		templates = map[string]string{
			"accepted":    "Rifeeraaliin keessan gara {{Hospital}} – {{Department}} fudhatameera. Gareen simatu beellamaaf isin quunnama. Lakk: {{ReferralID}}",
			"scheduled":   "Beellamni keessan {{Date}} irratti {{Hospital}}, {{Department}} dha. Maaloo yeroon dhiyaadhaa. Lakk: {{ReferralID}}",
			"missed":      "Beellama keessan {{Date}} irratti {{Hospital}} qabdan dabarsitaniittu. Maaloo saffisaan hospitaala deemaa. Rifeeraaliin keessan irra deebi'amee ilaalamutti jira. Lakk: {{ReferralID}}",
			"rescheduled": "Beellamni keessan gara {{NewDate}} irratti {{Hospital}}, {{Department}} tti jijjiirameera. Beellamni duraa haqameera. Lakk: {{ReferralID}}",
			"reminder":    "Yaadachiisa: Boru sa'aatii {{Time}} irratti {{Hospital}} beellama qabdu. Maaloo waraqaa eenyummaa keessan fidaa. Lakk: {{ReferralID}}",
		}
	default:
		templates = map[string]string{
			"accepted":    "Your referral to {{Hospital}} – {{Department}} has been ACCEPTED. The receiving team will contact you for scheduling. Ref: {{ReferralID}}",
			"scheduled":   "Your appointment is on {{Date}} at {{Hospital}}, {{Department}}. Please arrive on time. Ref: {{ReferralID}}",
			"missed":      "You missed your appointment on {{Date}} at {{Hospital}}. Please visit the hospital for a clinical update as soon as possible. Your referral is being re-evaluated. Ref: {{ReferralID}}",
			"rescheduled": "Your appointment has been RESCHEDULED to {{NewDate}} at {{Hospital}}, {{Department}}. Previous appointment cancelled. Ref: {{ReferralID}}",
			"reminder":    "Reminder: You have an appointment at {{Hospital}} tomorrow at {{Time}}. Please bring your ID. Ref: {{ReferralID}}",
		}
	}

	tmpl, exists := templates[templateKey]
	if !exists {
		tmpl = templates["scheduled"] // fallback
	}

	for k, v := range placeholders {
		tmpl = strings.ReplaceAll(tmpl, "{{"+k+"}}", v)
	}
	return tmpl
}
