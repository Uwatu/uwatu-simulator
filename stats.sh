#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  UWATU DASHBOARD v7.1 // HYPER-CYBER COMMITTER EDITION
#  macOS / zsh / bash 3 compatible
# ═══════════════════════════════════════════════════════════════

P='\033[38;5;135m'
C='\033[38;5;51m'
G='\033[38;5;82m'
R='\033[38;5;196m'
Y='\033[38;5;226m'
O='\033[38;5;208m'
M='\033[38;5;213m'
T='\033[38;5;87m'
W='\033[1;37m'
D='\033[38;5;244m'
B='\033[38;5;33m'
NC='\033[0m'
BOLD='\033[1m'

# ── ARG PARSING ──────────────────────────────────────────────
SHOW_ALL=false
SINCE_DAYS=30
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --all)    SHOW_ALL=true ;;
        --days)   SINCE_DAYS="$2"; shift ;;
        --branch) BRANCH="$2"; shift ;;
        --help|-h)
            echo -e "${P}UWATU DASHBOARD v7.1${NC}"
            echo -e "  ${W}--all${NC}         All-time stats"
            echo -e "  ${W}--days N${NC}      Stats for last N days"
            echo -e "  ${W}--branch NAME${NC} Specific branch"
            exit 0 ;;
    esac
    shift
done

if ! git rev-parse --is-inside-work-tree &>/dev/null; then
    echo -e "${R}✖ Not inside a git repository.${NC}"; exit 1
fi

if $SHOW_ALL; then
    SINCE_ARG=""
    WINDOW_LABEL="ALL TIME"
else
    SINCE_DATE=$(date -v-"${SINCE_DAYS}"d +%Y-%m-%d 2>/dev/null || date -d "${SINCE_DAYS} days ago" +%Y-%m-%d)
    SINCE_ARG="--since=${SINCE_DATE}"
    WINDOW_LABEL="LAST ${SINCE_DAYS} DAYS"
fi

clear

# ── HEADER ───────────────────────────────────────────────────
echo -e "${P}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${P}║${W}${BOLD}  UWATU DASHBOARD v7.1  ${D}//  ${C}HYPER-CYBER COMMITTER EDITION${P}       ║${NC}"
echo -e "${P}╚══════════════════════════════════════════════════════════════╝${NC}"

REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
NOW=$(date '+%Y-%m-%d %H:%M')
echo -e "  ${D}repo:${NC} ${W}${REPO_NAME}${NC}  ${D}│${NC}  ${D}branch:${NC} ${C}${BRANCH}${NC}  ${D}│${NC}  ${D}window:${NC} ${Y}${WINDOW_LABEL}${NC}  ${D}│${NC}  ${D}at:${NC} ${D}${NOW}${NC}"

# ── 1. CORE METRICS ──────────────────────────────────────────
TOTAL_COMMITS=$(git rev-list --count HEAD $SINCE_ARG 2>/dev/null || echo 0)
STAGED=$(git diff --cached --name-only 2>/dev/null | wc -l | xargs)
UNSTAGED=$(git diff --name-only 2>/dev/null | wc -l | xargs)
UNTRACKED=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l | xargs)
CONTRIBUTORS=$(git log --format='%ae' $SINCE_ARG | sort -u | wc -l | xargs)
TAGS=$(git tag | wc -l | xargs)
BRANCHES=$(git branch | wc -l | xargs)
FIRST_COMMIT=$(git log --reverse --format='%ad' --date=short $SINCE_ARG | head -n1)
LAST_COMMIT=$(git log -1 --format='%ad' --date=short $SINCE_ARG 2>/dev/null)

echo -e ""
echo -e "${P}  ┌─[ CORE METRICS ]${NC}"
echo -e "${P}  │${NC}  ${W}Commits:${NC} ${C}${TOTAL_COMMITS}${NC}  ${D}│${NC}  ${W}Contributors:${NC} ${C}${CONTRIBUTORS}${NC}  ${D}│${NC}  ${W}Branches:${NC} ${C}${BRANCHES}${NC}  ${D}│${NC}  ${W}Tags:${NC} ${C}${TAGS}${NC}"
echo -e "${P}  │${NC}  ${G}Staged: ${STAGED}${NC}  ${D}│${NC}  ${R}Unstaged: ${UNSTAGED}${NC}  ${D}│${NC}  ${D}Untracked: ${UNTRACKED}${NC}"
echo -e "${P}  │${NC}  ${D}First:${NC} ${D}${FIRST_COMMIT}${NC}  ${D}→${NC}  ${D}Last:${NC} ${D}${LAST_COMMIT}${NC}"
echo -e "${P}  └${NC}"

# ── 2. COMMITTER LEADERBOARD ─────────────────────────────────
# POSIX-safe: no mapfile, no declare -A, no octal 08/09
echo -e "\n${P}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${P}║${W}${BOLD}  👤  COMMITTER LEADERBOARD                                     ${P}║${NC}"
echo -e "${P}╚═══════════════════════════════════════════════════════════════╝${NC}"

TMPDIR_DASH=$(mktemp -d)
AUTHOR_LIST="$TMPDIR_DASH/authors.txt"
git log --format='%an' $SINCE_ARG | sort -u > "$AUTHOR_LIST"

# Find max commits for bar scaling
MAX_AC=1
while IFS= read -r AUTHOR; do
    AC=$(git log --format='%an' $SINCE_ARG | grep -Fc "$AUTHOR" || echo 0)
    [ "$AC" -gt "$MAX_AC" ] && MAX_AC=$AC
done < "$AUTHOR_LIST"

COLORS=("$C" "$G" "$Y" "$O" "$M" "$T" "$B")
NUM_COLORS=${#COLORS[@]}
IDX=0
RANK=1

while IFS= read -r AUTHOR; do
    AColor="${COLORS[$((IDX % NUM_COLORS))]}"

    AC=$(git log --format='%an' $SINCE_ARG | grep -Fc "$AUTHOR" || echo 0)

    STATS=$(git log --author="$AUTHOR" --numstat --pretty=format:"" $SINCE_ARG 2>/dev/null | \
        awk '/^[0-9]/ && $1 != "-" && $2 != "-" { ins += $1; del += $2; files[$3]=1 }
             END { print ins+0, del+0, length(files) }')
    A_INS=$(echo "$STATS" | awk '{print $1}')
    A_DEL=$(echo "$STATS" | awk '{print $2}')
    A_FILES=$(echo "$STATS" | awk '{print $3}')
    A_NET=$((A_INS - A_DEL))

    A_FIRST=$(git log --author="$AUTHOR" --format='%ad' --date=short $SINCE_ARG | tail -n1)
    A_LAST=$(git log  --author="$AUTHOR" --format='%ad' --date=short $SINCE_ARG | head -n1)

    A_PEAK=$(git log --author="$AUTHOR" --format='%ad' --date='format:%H' $SINCE_ARG | \
        awk '{h=int($1)+0; c[h]++} END{mx=0; for(h in c) if(c[h]>mx){mx=c[h];ph=h}; printf "%02d:00",ph}')

    A_TOP=$(git log --author="$AUTHOR" --pretty=format: --name-only $SINCE_ARG | \
        grep -v '^$' | sort | uniq -c | sort -nr | head -n1 | awk '{print $2}')

    BAR_LEN=$(( AC * 30 / MAX_AC ))
    [ "$BAR_LEN" -lt 1 ] && [ "$AC" -gt 0 ] && BAR_LEN=1

    if [ "$A_NET" -ge 0 ]; then NET_COLOR=$G; NET_SIGN="+"; else NET_COLOR=$R; NET_SIGN=""; fi

    TOTAL_LINES=$((A_INS + A_DEL))
    if [ "$TOTAL_LINES" -gt 0 ]; then
        I_FRAC=$(( A_INS * 30 / TOTAL_LINES ))
        D_FRAC=$(( 30 - I_FRAC ))
        PCT_ADD=$(( A_INS * 100 / TOTAL_LINES ))
        PCT_DEL=$(( A_DEL * 100 / TOTAL_LINES ))
    else
        I_FRAC=0; D_FRAC=30; PCT_ADD=0; PCT_DEL=0
    fi

    case $RANK in
        1) MEDAL="🥇" ;;
        2) MEDAL="🥈" ;;
        3) MEDAL="🥉" ;;
        *) MEDAL="   " ;;
    esac

    echo -e "${P}  ┌─────────────────────────────────────────────────────────────${NC}"
    printf "  ${AColor}${BOLD}%s  %-28s${NC}  ${D}rank #%d${NC}\n" "$MEDAL" "$AUTHOR" "$RANK"

    # Commit bar
    printf "  ${W}  Commits  :${NC} ${AColor}%-5s${NC} [" "$AC"
    for ((i=0; i<BAR_LEN; i++));  do printf "${AColor}█${NC}"; done
    for ((i=BAR_LEN; i<30; i++)); do printf "${D}░${NC}"; done
    printf "]\n"

    # Lines summary
    printf "  ${W}  Lines    :${NC} ${G}+%-8s${NC}  ${R}-%-8s${NC}  ${NET_COLOR}net %s%s${NC}\n" \
           "$A_INS" "$A_DEL" "$NET_SIGN" "$A_NET"

    # Churn bar
    printf "  ${W}  Churn    :${NC} ["
    for ((i=0; i<I_FRAC; i++));  do printf "${G}▓${NC}"; done
    for ((i=0; i<D_FRAC; i++));  do printf "${R}░${NC}"; done
    printf "]  ${D}%d%% add / %d%% del${NC}\n" "$PCT_ADD" "$PCT_DEL"

    printf "  ${W}  Files    :${NC} ${C}%s unique${NC}  ${D}│${NC}  ${W}Top file:${NC}  ${D}%s${NC}\n" \
           "$A_FILES" "${A_TOP:-(none)}"
    printf "  ${W}  Active   :${NC} ${D}%s → %s${NC}  ${D}│${NC}  ${W}Peak hour:${NC}  ${Y}%s${NC}\n" \
           "$A_FIRST" "$A_LAST" "${A_PEAK:-(n/a)}"

    IDX=$((IDX + 1))
    RANK=$((RANK + 1))
done < "$AUTHOR_LIST"
echo -e "${P}  └─────────────────────────────────────────────────────────────${NC}"

# ── SUMMARY TABLE ─────────────────────────────────────────────
echo -e "\n${P}  ┌─[ LEADERBOARD SUMMARY TABLE ]${NC}"
printf "  ${D}  %-22s  %7s  %9s  %9s  %10s${NC}\n" "AUTHOR" "COMMITS" "ADDS" "DELS" "NET"
echo -e "  ${D}  ──────────────────────────────────────────────────────${NC}"

RANK=1
while IFS= read -r AUTHOR; do
    AC=$(git log --format='%an' $SINCE_ARG | grep -Fc "$AUTHOR" || echo 0)
    STATS=$(git log --author="$AUTHOR" --numstat --pretty=format:"" $SINCE_ARG 2>/dev/null | \
        awk '/^[0-9]/ && $1 != "-" && $2 != "-" { ins += $1; del += $2 }
             END { print ins+0, del+0 }')
    A_INS=$(echo "$STATS" | awk '{print $1}')
    A_DEL=$(echo "$STATS" | awk '{print $2}')
    A_NET=$((A_INS - A_DEL))
    case $RANK in 1) M="🥇";; 2) M="🥈";; 3) M="🥉";; *) M="  ";; esac
    if [ "$A_NET" -ge 0 ]; then NC2=$G; NS="+"; else NC2=$R; NS=""; fi
    printf "  %s ${W}%-22s${NC}  ${C}%7s${NC}  ${G}%9s${NC}  ${R}%9s${NC}  ${NC2}%10s${NC}\n" \
           "$M" "$AUTHOR" "$AC" "+$A_INS" "-$A_DEL" "${NS}${A_NET}"
    RANK=$((RANK + 1))
done < "$AUTHOR_LIST"
echo -e "${P}  └${NC}"

rm -rf "$TMPDIR_DASH"

# ── 3. GLOBAL CODE CHURN ─────────────────────────────────────
echo -e "\n${P}  [ CODE CHURN — GLOBAL ]${NC}"
eval $(git log --shortstat --pretty=format:"" $SINCE_ARG | awk '
  /,/ {
    for (i=1; i<=NF; i++) {
      if ($i ~ /insertion/) ins += $(i-1)
      if ($i ~ /deletion/)  del += $(i-1)
    }
  }
  END { printf "INS=%d; DEL=%d", ins, del }')

TOTAL_CHURN=$((INS + DEL))
if [ "$TOTAL_CHURN" -gt 0 ]; then
    I_LEN=$((INS * 40 / TOTAL_CHURN))
    D_LEN=$((40 - I_LEN))
    printf "  ${G}+%-8s${NC} [" "$INS"
    for ((i=0; i<I_LEN; i++)); do printf "${G}█${NC}"; done
    for ((i=0; i<D_LEN; i++)); do printf "${R}█${NC}"; done
    printf "] ${R}-%-8s${NC}\n" "$DEL"
    NET_G=$((INS - DEL))
    if [ "$NET_G" -ge 0 ]; then echo -e "  ${D}Net delta:${NC} ${G}+${NET_G} lines${NC}"
    else echo -e "  ${D}Net delta:${NC} ${R}${NET_G} lines${NC}"; fi
fi

# ── 4. TECH STACK ────────────────────────────────────────────
echo -e "\n${P}  [ TECH STACK ]${NC}"
TOTAL_FILES=$(find . -type f -not -path '*/.*' -not -path '*/node_modules/*' -not -path '*/dist/*' | wc -l | xargs)
[ "$TOTAL_FILES" -eq 0 ] && TOTAL_FILES=1
find . -type f -not -path '*/.*' -not -path '*/node_modules/*' -not -path '*/dist/*' | \
    sed 's/.*\.//' | sort | uniq -c | sort -nr | head -n 7 | while read -r count lang; do
    pct=$((count * 100 / TOTAL_FILES))
    bar_size=$((pct / 3))
    [ "$bar_size" -lt 1 ] && [ "$pct" -gt 0 ] && bar_size=1
    printf "  ${C}%-10s${NC} ${D}│${NC} " "$lang"
    for ((i=0; i<bar_size; i++)); do printf "${W}━${NC}"; done
    printf " ${W}%d%%${NC} ${D}(%d files)${NC}\n" "$pct" "$count"
done

# ── 5. HOTSPOTS ──────────────────────────────────────────────
echo -e "\n${P}  [ HOTSPOTS — Most Edited Files ]${NC}"
git log --pretty=format: --name-only $SINCE_ARG | grep -v '^$' | grep -v 'node_modules' | \
sort | uniq -c | sort -nr | head -n 5 | \
awk -v C="$C" -v D="$D" -v NC="$NC" \
    'NR==1{medal="🔥"} NR==2{medal="⚡"} NR==3{medal="💡"} NR>3{medal="  "}
     {printf "  %s  %s%-4s edits%s  %s%s%s\n", medal, C, $1, NC, D, $2, NC}'

# ── 6. MOMENTUM ──────────────────────────────────────────────
echo -e "\n${P}  [ MOMENTUM — Last 7 Days ]${NC}"
MAX_DC=1
for i in 6 5 4 3 2 1 0; do
    DAY=$(date -v-"$i"d +%Y-%m-%d 2>/dev/null || date -d "$i days ago" +%Y-%m-%d)
    count=$(git log --oneline --since="$DAY 00:00" --until="$DAY 23:59" 2>/dev/null | wc -l | xargs)
    [ "$count" -gt "$MAX_DC" ] && MAX_DC=$count
done
for i in 6 5 4 3 2 1 0; do
    DAY=$(date -v-"$i"d +%Y-%m-%d 2>/dev/null || date -d "$i days ago" +%Y-%m-%d)
    DAYNAME=$(date -v-"$i"d +%a 2>/dev/null || date -d "$i days ago" +%a)
    count=$(git log --oneline --since="$DAY 00:00" --until="$DAY 23:59" 2>/dev/null | wc -l | xargs)
    BAR_W=$(( count * 25 / MAX_DC ))
    [ "$BAR_W" -lt 1 ] && [ "$count" -gt 0 ] && BAR_W=1
    printf "  ${D}%s (%s)${NC} │ " "$DAY" "$DAYNAME"
    if [ "$count" -gt 0 ]; then
        [ "$count" -eq "$MAX_DC" ] && COLOR=$Y || COLOR=$G
        for ((j=0; j<BAR_W; j++)); do printf "${COLOR}▮${NC}"; done
        printf " ${W}%s commits${NC}" "$count"
    else printf "${D}·  quiet${NC}"; fi
    echo ""
done

# ── 7. HOURLY HEATMAP (no octal bug) ─────────────────────────
echo -e "\n${P}  [ COMMIT HEATMAP — Hour of Day ]${NC}"
HEATMAP_FILE=$(mktemp)
git log --format='%ad' --date='format:%H' $SINCE_ARG | \
    awk '{h=int($1)+0; counts[h]++} END {for(h=0;h<24;h++) print h, counts[h]+0}' \
    > "$HEATMAP_FILE"
MAX_H=$(awk 'BEGIN{m=0}{if($2>m)m=$2}END{print m}' "$HEATMAP_FILE")
[ "$MAX_H" -eq 0 ] && MAX_H=1
printf "  "
while read -r h v; do
    intensity=$(( v * 4 / MAX_H ))
    case $intensity in
        0) printf "${D}·${NC}" ;;
        1) printf "${B}▪${NC}" ;;
        2) printf "${C}▪${NC}" ;;
        3) printf "${G}▪${NC}" ;;
        4) printf "${Y}█${NC}" ;;
    esac
done < "$HEATMAP_FILE"
echo ""
echo -e "  ${D}00    03    06    09    12    15    18    21    23  (UTC)${NC}"
rm -f "$HEATMAP_FILE"

# ── 8. RECENT COMMITS ────────────────────────────────────────
echo -e "\n${P}  [ RECENT COMMITS ]${NC}"
git log --format="%h|%an|%ad|%s" --date=short $SINCE_ARG | head -n 8 | \
while IFS='|' read -r hash author date subject; do
    printf "  ${D}%s${NC}  ${C}%-18s${NC}  ${D}%s${NC}  %s\n" \
        "$hash" "${author:0:18}" "$date" "${subject:0:55}"
done

# ── 9. STALE BRANCHES ────────────────────────────────────────
echo -e "\n${P}  [ STALE BRANCHES — No commits 14+ days ]${NC}"
STALE=0
NOW_TS=$(date +%s)
while IFS= read -r br; do
    LAST_TS=$(git log -1 --format="%ct" "$br" 2>/dev/null)
    [ -z "$LAST_TS" ] && continue
    AGE=$(( (NOW_TS - LAST_TS) / 86400 ))
    if [ "$AGE" -ge 14 ]; then
        LAST_REL=$(git log -1 --format="%ar" "$br" 2>/dev/null)
        printf "  ${R}%-35s${NC}  ${D}last: %s${NC}\n" "$br" "$LAST_REL"
        STALE=$((STALE + 1))
    fi
done < <(git branch --format='%(refname:short)' | grep -v "^${BRANCH}$")
[ "$STALE" -eq 0 ] && echo -e "  ${G}✓ All branches are active.${NC}"

# ── 10. PRODUCTIVITY INSIGHTS ────────────────────────────────
PEAK_LINE=$(git log --format='%ad' --date='format:%H:00' $SINCE_ARG | sort | uniq -c | sort -nr | head -n1)
PEAK_HOUR_VAL=$(echo "$PEAK_LINE" | awk '{print $2}')
PEAK_HOUR_CNT=$(echo "$PEAK_LINE" | awk '{print $1}')
AVG_PER_DAY=$(awk "BEGIN{printf \"%.1f\", $TOTAL_COMMITS/7}")
CHURN_RATIO=$(awk "BEGIN{printf \"%d\", ($INS>0 ? $DEL*100/$INS : 0)}")

echo -e "\n${P}  [ PRODUCTIVITY INSIGHTS ]${NC}"
echo -e "  ${Y}⚡ Peak Commit Hour  :${NC} ${W}${PEAK_HOUR_VAL}${NC}  ${D}(${PEAK_HOUR_CNT} commits)${NC}"
[ "$TOTAL_COMMITS" -gt 0 ] && \
    echo -e "  ${Y}⚡ Lines / Commit    :${NC} ${W}$(( (INS + DEL) / TOTAL_COMMITS ))${NC}"
echo -e "  ${Y}⚡ Avg Commits / Day :${NC} ${W}${AVG_PER_DAY}${NC}"
echo -e "  ${Y}⚡ Churn Ratio       :${NC} ${W}${CHURN_RATIO}%${NC}  ${D}(dels / adds)${NC}"
echo -e "  ${Y}⚡ Files in Repo     :${NC} ${W}${TOTAL_FILES}${NC}"

NET_TOTAL=$((INS - DEL))
if   [ "$TOTAL_COMMITS" -eq 0 ];   then STATUS="NO ACTIVITY";           COLOR=$D
elif [ "$NET_TOTAL" -gt 5000 ];    then STATUS="EXPLOSIVE GROWTH  🚀";   COLOR=$G
elif [ "$NET_TOTAL" -gt 500 ];     then STATUS="ACTIVE GROWTH  ⬆️";      COLOR=$C
elif [ "$NET_TOTAL" -lt -2000 ];   then STATUS="HEAVY REFACTOR  🔧";     COLOR=$O
elif [ "$NET_TOTAL" -lt 0 ];       then STATUS="CLEANUP PHASE  🧹";      COLOR=$C
else                                    STATUS="STABLE DEVELOPMENT  ✅";  COLOR=$Y
fi

echo -e "\n  ${D}STATUS:${NC}  ${COLOR}${BOLD}${STATUS}${NC}"
echo -e "${P}══════════════════════════════════════════════════════════════${NC}"