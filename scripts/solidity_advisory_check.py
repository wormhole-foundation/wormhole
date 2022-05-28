import requests

url = "https://raw.githubusercontent.com/ethereum/solidity/develop/docs/bugs.json"

resp = requests.get(url=url)
bugs = resp.json()

# Known bug IDs
#
# When a new Solidity bug is released, the known_bugs list will need to be updated to stop alerting
#
known_bugs = [
	'SOL-2018-1',
	'SOL-2022-3',
	'SOL-2022-2',
	'SOL-2022-1',
	'SOL-2021-4',
	'SOL-2021-3',
	'SOL-2021-2',
	'SOL-2021-1',
	'SOL-2020-11',
	'SOL-2020-10',
	'SOL-2020-9',
	'SOL-2020-8',
	'SOL-2020-7',
	'SOL-2020-6',
	'SOL-2020-5',
	'SOL-2020-4',
	'SOL-2020-3',
	'SOL-2020-1',
	'SOL-2020-2',
	'SOL-2020-1',
	'SOL-2019-10',
	'SOL-2019-9',
	'SOL-2019-8',
	'SOL-2019-7',
	'SOL-2019-6',
	'SOL-2019-5',
	'SOL-2019-5',
	'SOL-2019-4',
	'SOL-2019-4',
	'SOL-2019-3',
	'SOL-2019-3',
	'SOL-2019-2',
	'SOL-2019-1',
	'SOL-2018-4',
	'SOL-2018-3',
	'SOL-2018-2',
	'SOL-2018-1',
	'SOL-2017-5',
	'SOL-2017-4',
	'SOL-2017-3',
	'SOL-2017-2',
	'SOL-2017-1',
	'SOL-2016-11',
	'SOL-2016-10',
	'SOL-2016-9',
	'SOL-2016-8',
	'SOL-2016-7',
	'SOL-2016-6',
	'SOL-2016-5',
	'SOL-2016-4',
	'SOL-2016-3',
	'SOL-2016-2',
	'SOL-2016-1',
]

new_bug = False

for bug in bugs:
	if bug['uid'] not in known_bugs:
		new_bug = False

		# TODO: Turn this into a real-time SIEM event for triage/action
		print("[+] New Solidity Bug!!!")
		print("UID: {0}".format(bug['uid']))
		print("Name: {0}".format(bug['name']))
		print("Summary: {0}".format(bug['summary']))
		print("Description: {0}".format(bug['description']))
		print("Severity: {0}".format(bug['severity']))
		print("Fixed: {0}".format(bug['fixed']))

if new_bug:
	exit(1)
else:
    exit(0)