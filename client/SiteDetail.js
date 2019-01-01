/** @format */

import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars

const fetch = window.fetch

import Data from './Data'
import log from './log'
import {encodedate} from './helpers'

export default function SiteDetail({domain}) {
  let [period, setPeriod] = useState({
    ending: encodedate(new Date()),
    interval: 45
  })

  let [today, setToday] = useState({})
  let [days, setDays] = useState({days: [], stats: [], compendium: {}})
  let [months, setMonths] = useState({months: [], compendium: {}})

  useEffect(
    () => {
      queryToday(domain).then(setToday)

      if (period.interval <= 90) {
        queryDays(domain, period.interval).then(setDays)
      } else {
        queryMonths(domain, parseInt(period.interval / 30)).then(setMonths)
      }
    },
    [period]
  )

  let usingMonths = period.interval > 90

  return (
    <div className="container">
      <div className="content">
        <h4 className="title is-3">{domain}</h4>
      </div>
      <Data
        today={today}
        days={days}
        months={months}
        usingMonths={usingMonths}
        period={period}
        updateInterval={v => setPeriod({...period, interval: v})}
      />
    </div>
  )
}

async function queryDays(domain, nlastdays) {
  try {
    let res = await fetch('/query/days', {
      method: 'POST',
      body: JSON.stringify({
        domain,
        last: nlastdays
      }),
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json'
      }
    })

    if (!res.ok) throw new Error(await res.text())
    let {days, stats, compendium} = await res.json()

    return {
      days,
      stats,
      compendium
    }
  } catch (e) {
    log.error(e)
  }
}

async function queryMonths(domain, nlastmonths) {
  try {
    let res = await fetch('/query/months', {
      method: 'POST',
      body: JSON.stringify({
        domain,
        last: nlastmonths
      }),
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json'
      }
    })

    if (!res.ok) throw new Error(await res.text())
    let {months, compendium} = await res.json()

    return {months, compendium}
  } catch (e) {
    log.error(e)
  }
}

async function queryToday(domain) {
  try {
    let res = await fetch('/query/today', {
      method: 'POST',
      body: JSON.stringify({
        domain
      }),
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json'
      }
    })

    if (!res.ok) throw new Error(await res.text())
    let stats = await res.json()

    return stats
  } catch (e) {
    log.error(e)
  }
}
