import time
import requests


def main():
    for site in sites:
        hosts = {}
        r = requests.get(
            COUCH + "/_all_docs",
            params={"start_key": site, "end_key": site + "~", "include_docs": True},
        )
        docs = [row["doc"] for row in r.json()["rows"]]
        for doc in docs:
            print(doc)
            time.sleep(1)


sites = [
    "l4mkc824",
    "qvjlfnlo",
    "l984a9px",
    "qv2oun8n",
    "xjkytrxz",
    "9ykvs7rk",
    "ql59s6ky",
    "xo68uqz1",
    "kmv6tp0n",
    "qq26croo",
    "8xl5i644",
    "91f94p",
    "8xolamo6",
    "3x3mb94v",
    "6xf9no",
    "vv1rh3x4",
    "y3flv3",
    "ppzqsrrr",
    "kofm77",
    "pp2lt6lp",
    "l42ru9po",
    "pm1ntxrn",
    "0p58tr48",
    "yvrka3r5",
    "2vyxan66",
    "2w81ikl5",
    "nrf4rr",
    "xjkytv3y",
    "mzfzx9",
    "3vfpw1",
    "p9fq25",
    "z6m0h296",
    "yklzb7m1",
    "mzxqavpw",
    "yklzbk2m",
    "pm1ntlom",
    "lrfp90",
    "5x5obk18",
    "91o2i47k",
    "zlwxtxyw",
    "5wk2h8vr",
]

COUCH = "http://trackingcode-backend:9384hskdjbf3bu3sd@45.77.79.192:5984/trackingcode"

main()
