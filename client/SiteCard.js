const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')

const graphql = require('./graphql')

module.exports = React.createClass({
  getInitialState () {
    return {
      site: {}
    }
  },

  q: `
    query siteOverview($code: String!) {
      site(code: $code) {
        name
        code
        days(last:7) {
          day
          s
          v
        }
      }
    }
  `,

  query () {
    graphql.query(this.q, {code: this.props.code})
    .then(r => this.setState(r))
    .catch(console.log.bind(console))
  },

  componentDidMount () {
    this.query()
  },

  componentWillReceiveProps (nextProps) {
    this.setState({code: nextProps.code}, this.query)
  },

  render () {
    return h('.card', {id: this.props.code}, [
      h('.card-content', [
        h('h4.title.is-4', this.state.site.name),
        h('h6.subtitle.is-6', this.state.site.code)
      ]),
      h('.card-image', [
        h('figure.image', [
          h(R.ResponsiveContainer, {height: 200, width: '100%'}, [
            h(R.ComposedChart, {data: this.state.site.days}, [
              h(R.Bar, {
                dataKey: 's',
                fill: '#8884d8',
                isAnimationActive: false
              }),
              h(R.Line, {
                dataKey: 'v',
                stroke: '#82ca9d',
                isAnimationActive: false
              })
            ])
          ])
        ])
      ])
    ])
  }
})
