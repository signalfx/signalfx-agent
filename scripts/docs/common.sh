
# Use the marketing name in product docs more but less in this repo since this
# is more developer-oriented.
use_marketing_name() {
  sed -e 's/\([tT]\)he agent/\1he Smart Agent/g'
}

