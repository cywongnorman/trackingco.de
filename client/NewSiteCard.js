const React = require('react')
const withClickOutside = require('react-click-outside')
const h = require('react-hyperscript')

const log = require('./log')
const graphql = require('./graphql')

module.exports = withClickOutside(React.createClass({
  getInitialState () {
    return {
      name: '',
      writing: false,
      waiting: false
    }
  },

  create () {
    window.tc && window.tc(4)
    graphql.mutate(`
      ($name: String!) {
        createSite(name: $name) {
          code
        }
      }
    `, {name: this.state.name})
    .then(r => {
      log.success(this.state.name, 'created! You can start tracking it.')
      this.props.onNewSiteCreated()
      this.setState(this.getInitialState())
    })
    .catch(log.error)
  },

  render () {
    return h('.card.newsite', {id: this.props.code}, [
      h('.card-content', {onClick: this.startWriting}, [
        h('h4.title.is-5', this.state.writing ? 'give a name to the site' : 'add a new site'),
        this.state.writing
        ? h('.card-content', [
          h('form', {onSubmit: this.submit}, [
            h('input.input', {
              onChange: this.write,
              value: this.state.name,
              ref: el => el && el.focus()
            }),
            h('button.button.is-primary', {
              className: this.state.waiting ? 'is-loading' : ''
            }, 'add')
          ])
        ])
        : h('i.fa.fa-plus')
      ])
    ])
  },

  startWriting () {
    this.setState({writing: true})
  },

  write (e) {
    this.setState({name: e.target.value})
  },

  submit (e) {
    e.preventDefault()
    this.setState({waiting: true})
    this.create()
  },

  handleClickOutside () {
    this.setState({writing: false})
  }
}))
