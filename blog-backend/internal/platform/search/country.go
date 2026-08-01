package search

import "strings"

// tavilyCountries maps ISO 3166-1 alpha-2 codes onto the English country names
// Tavily's `country` enum accepts. Only the codes Tavily supports are listed;
// anything else is dropped from the request rather than rejected.
var tavilyCountries = map[string]string{
	"AF": "afghanistan", "AL": "albania", "DZ": "algeria", "AD": "andorra", "AO": "angola",
	"AR": "argentina", "AM": "armenia", "AU": "australia", "AT": "austria", "AZ": "azerbaijan",
	"BS": "bahamas", "BH": "bahrain", "BD": "bangladesh", "BB": "barbados", "BY": "belarus",
	"BE": "belgium", "BZ": "belize", "BJ": "benin", "BT": "bhutan", "BO": "bolivia",
	"BA": "bosnia and herzegovina", "BW": "botswana", "BR": "brazil", "BN": "brunei", "BG": "bulgaria",
	"BF": "burkina faso", "BI": "burundi", "KH": "cambodia", "CM": "cameroon", "CA": "canada",
	"CV": "cape verde", "CF": "central african republic", "TD": "chad", "CL": "chile", "CN": "china",
	"CO": "colombia", "KM": "comoros", "CG": "congo", "CR": "costa rica", "HR": "croatia",
	"CU": "cuba", "CY": "cyprus", "CZ": "czech republic", "DK": "denmark", "DJ": "djibouti",
	"DO": "dominican republic", "EC": "ecuador", "EG": "egypt", "SV": "el salvador",
	"GQ": "equatorial guinea", "ER": "eritrea", "EE": "estonia", "ET": "ethiopia", "FJ": "fiji",
	"FI": "finland", "FR": "france", "GA": "gabon", "GM": "gambia", "GE": "georgia",
	"DE": "germany", "GH": "ghana", "GR": "greece", "GT": "guatemala", "GN": "guinea",
	"HT": "haiti", "HN": "honduras", "HU": "hungary", "IS": "iceland", "IN": "india",
	"ID": "indonesia", "IR": "iran", "IQ": "iraq", "IE": "ireland", "IL": "israel",
	"IT": "italy", "JM": "jamaica", "JP": "japan", "JO": "jordan", "KZ": "kazakhstan",
	"KE": "kenya", "KW": "kuwait", "KG": "kyrgyzstan", "LV": "latvia", "LB": "lebanon",
	"LS": "lesotho", "LR": "liberia", "LY": "libya", "LI": "liechtenstein", "LT": "lithuania",
	"LU": "luxembourg", "MG": "madagascar", "MW": "malawi", "MY": "malaysia", "MV": "maldives",
	"ML": "mali", "MT": "malta", "MR": "mauritania", "MU": "mauritius", "MX": "mexico",
	"MD": "moldova", "MC": "monaco", "MN": "mongolia", "ME": "montenegro", "MA": "morocco",
	"MZ": "mozambique", "MM": "myanmar", "NA": "namibia", "NP": "nepal", "NL": "netherlands",
	"NZ": "new zealand", "NI": "nicaragua", "NE": "niger", "NG": "nigeria", "KP": "north korea",
	"MK": "north macedonia", "NO": "norway", "OM": "oman", "PK": "pakistan", "PA": "panama",
	"PG": "papua new guinea", "PY": "paraguay", "PE": "peru", "PH": "philippines", "PL": "poland",
	"PT": "portugal", "QA": "qatar", "RO": "romania", "RU": "russia", "RW": "rwanda",
	"SA": "saudi arabia", "SN": "senegal", "RS": "serbia", "SG": "singapore", "SK": "slovakia",
	"SI": "slovenia", "SO": "somalia", "ZA": "south africa", "KR": "south korea", "SS": "south sudan",
	"ES": "spain", "LK": "sri lanka", "SD": "sudan", "SE": "sweden", "CH": "switzerland",
	"SY": "syria", "TW": "taiwan", "TJ": "tajikistan", "TZ": "tanzania", "TH": "thailand",
	"TG": "togo", "TT": "trinidad and tobago", "TN": "tunisia", "TR": "turkey", "TM": "turkmenistan",
	"UG": "uganda", "UA": "ukraine", "AE": "united arab emirates", "GB": "united kingdom",
	"US": "united states", "UY": "uruguay", "UZ": "uzbekistan", "VE": "venezuela", "VN": "vietnam",
	"YE": "yemen", "ZM": "zambia", "ZW": "zimbabwe",
}

// tavilyCountry translates an ISO alpha-2 code into Tavily's country name.
func tavilyCountry(code string) (string, bool) {
	name, ok := tavilyCountries[strings.ToUpper(strings.TrimSpace(code))]
	return name, ok
}
