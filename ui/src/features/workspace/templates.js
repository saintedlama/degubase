// Hex values drawn from palette 0 (Set 1) and palette 1 (Set 2) in palettes.js
// so select chip colours feel native alongside user-created columns.
const C = {
  red:    '#e41a1c',
  blue:   '#377eb8',
  green:  '#4daf4a',
  purple: '#984ea3',
  orange: '#ff7f00',
  brown:  '#a65628',
  pink:   '#f781bf',
  teal:   '#66c2a5',
  yellow: '#e6ab02',
  gray:   '#b3b3b3',
}

function sel(choices) {
  return {
    choices: choices.map(c => c[0]),
    choiceColors: Object.fromEntries(choices.map(c => [c[0], c[1]])),
    palette: 0,
  }
}

function col(name, type, options) {
  return options ? { name, type, options } : { name, type }
}

function view(name, type, config = {}, isDefault = false) {
  return { name, type, config, isDefault }
}

// Shared option sets reused across templates
const STATUS_TASK = sel([
  ['Backlog',     C.gray  ],
  ['In Progress', C.blue  ],
  ['In Review',   C.orange],
  ['Done',        C.green ],
  ['Blocked',     C.red   ],
])

const PRIORITY = sel([
  ['Low',      C.teal  ],
  ['Medium',   C.yellow],
  ['High',     C.orange],
  ['Critical', C.red   ],
])

export const TEMPLATE_GROUPS = [
  {
    label: 'Standard',
    templates: [
      {
        id: 'task-tracker',
        name: 'Task Tracker',
        description: 'Lightweight task board with Kanban workflow.',
        icon: '✅',
        columns: [
          col('Task', 'text'),
          col('Owner', 'text'),
          col('Status', 'single-select', STATUS_TASK),
          col('Priority', 'single-select', PRIORITY),
          col('Due Date', 'date'),
          col('Notes', 'markdown'),
        ],
        views: [
          view('Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'crm',
        name: 'CRM',
        description: 'Track contacts, companies, and deal stages.',
        icon: '👥',
        columns: [
          col('Name', 'text'),
          col('Company', 'text'),
          col('Email', 'email'),
          col('Phone', 'text'),
          col('Stage', 'single-select', sel([
            ['Lead',        C.gray  ],
            ['Qualified',   C.blue  ],
            ['Proposal',    C.orange],
            ['Negotiation', C.yellow],
            ['Closed Won',  C.green ],
            ['Closed Lost', C.red   ],
          ])),
          col('Last Contact', 'date'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Stages', 'kanban', { xCol: 'Stage' }),
        ],
      },
      {
        id: 'bug-tracker',
        name: 'Bug Tracker',
        description: 'Capture, triage, and resolve software defects.',
        icon: '🐛',
        columns: [
          col('Title', 'text'),
          col('Severity', 'single-select', PRIORITY),
          col('Status', 'single-select', sel([
            ['Open',        C.red   ],
            ['In Progress', C.orange],
            ['Fixed',       C.green ],
            ["Won't Fix",   C.gray  ],
            ['Duplicate',   C.purple],
          ])),
          col('Assignee', 'text'),
          col('Reported', 'date'),
          col('Steps to Reproduce', 'markdown'),
          col('Screenshot', 'image'),
          col('Attachments', 'file'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Triage Board', 'kanban', { xCol: 'Status' }, true),
          view('Severity Matrix', 'matrix', { xCol: 'Status', yCol: 'Severity' }),
        ],
      },
      {
        id: 'content-calendar',
        name: 'Content Calendar',
        description: 'Plan posts, newsletters, and publishing schedules.',
        icon: '📅',
        columns: [
          col('Title', 'text'),
          col('Channel', 'single-select', sel([
            ['Blog',       C.blue  ],
            ['Newsletter', C.orange],
            ['LinkedIn',   C.teal  ],
            ['X',          C.gray  ],
            ['Instagram',  C.pink  ],
            ['YouTube',    C.red   ],
          ])),
          col('Status', 'single-select', sel([
            ['Idea',      C.gray  ],
            ['Draft',     C.blue  ],
            ['In Review', C.orange],
            ['Scheduled', C.yellow],
            ['Published', C.green ],
          ])),
          col('Author', 'text'),
          col('Publish Date', 'date'),
          col('Brief', 'markdown'),
        ],
        views: [
          view('Editorial Board', 'kanban', { xCol: 'Status' }, true),
          view('Timeline', 'timeline', { dateCol: 'Publish Date' }),
        ],
      },
    ],
  },
  {
    label: 'Product & Team',
    templates: [
      {
        id: 'roadmap',
        name: 'Product Roadmap',
        description: 'Now / Next / Later roadmap with target releases and outcomes.',
        icon: '🗺️',
        columns: [
          col('Project', 'text'),
          col('Horizon', 'single-select', sel([
            ['Now',     C.green ],
            ['Next',    C.blue  ],
            ['Later',   C.gray  ],
            ['Shipped', C.teal  ],
            ['Paused',  C.orange],
          ])),
          col('Target Release', 'text'),
          col('Lead', 'text'),
          col('Why / Outcome', 'text'),
          col('Target Date', 'date'),
          col('Spec / Link', 'url'),
          col('Category', 'single-select', sel([
            ['Core Feature',      C.blue  ],
            ['UX & Polish',       C.purple],
            ['Growth',            C.green ],
            ['Infra & Tech Debt', C.orange],
          ])),
          col('Notes', 'markdown'),
        ],
        views: [
          view('Roadmap Board', 'kanban', { xCol: 'Horizon' }, true),
          view('Release Timeline', 'timeline', { dateCol: 'Target Date' }),
        ],
      },
      {
        id: 'goals-and-bets',
        name: 'Goals & Bets',
        description: 'Pragmatic quarterly bets and target metrics in a single table.',
        icon: '🎯',
        columns: [
          col('Goal / Bet', 'text'),
          col('Period', 'single-select', sel([
            ['Q1', C.teal], ['Q2', C.blue], ['Q3', C.orange], ['Q4', C.purple], ['Ongoing', C.gray],
          ])),
          col('Status', 'single-select', sel([
            ['On Track',    C.green ],
            ['Needs Focus', C.yellow],
            ['At Risk',     C.red   ],
            ['Achieved',    C.teal  ],
            ['Dropped',     C.gray  ],
          ])),
          col('Target Metric', 'text'),
          col('Current Progress', 'percent'),
          col('Lead', 'text'),
          col('Key Steps', 'checklist'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Status Board', 'kanban', { xCol: 'Status' }),
        ],
      },
      {
        id: 'feature-requests',
        name: 'Feature Requests',
        description: 'Capture user feedback, pain points, and feature demand.',
        icon: '💡',
        columns: [
          col('Title', 'text'),
          col('Status', 'single-select', sel([
            ['Under Review', C.gray  ],
            ['Planned',      C.blue  ],
            ['In Progress',  C.orange],
            ['Done',         C.green ],
            ['Declined',     C.red   ],
          ])),
          col('Demand', 'single-select', sel([
            ['High (Multiple Users)', C.red   ],
            ['Moderate',              C.orange],
            ['Nice to have',          C.teal  ],
          ])),
          col('Type', 'single-select', sel([
            ['Feature',      C.blue  ],
            ['UX / Polish',  C.purple],
            ['Integration',  C.teal  ],
            ['Performance',  C.yellow],
          ])),
          col('Customer / Source', 'text'),
          col('Pain Point', 'long-text'),
          col('Linked Project', 'text'),
        ],
        views: [
          view('Pipeline Board', 'kanban', { xCol: 'Status' }, true),
          view('Demand Matrix', 'matrix', { xCol: 'Status', yCol: 'Demand' }),
        ],
      },
      {
        id: 'changelog',
        name: 'Releases & Changelog',
        description: 'Plan releases, track changelog items, and note breaking changes.',
        icon: '📦',
        columns: [
          col('Version', 'text'),
          col('Status', 'single-select', sel([
            ['Draft',    C.gray  ],
            ['RC',       C.orange],
            ['Released', C.green ],
          ])),
          col('Release Date', 'date'),
          col('Highlights', 'markdown'),
          col('Breaking Changes', 'checkbox'),
          col('Release URL', 'url'),
        ],
        views: [
          view('Releases Board', 'kanban', { xCol: 'Status' }, true),
          view('Release Timeline', 'timeline', { dateCol: 'Release Date' }),
        ],
      },
      {
        id: 'incident-log',
        name: 'Incident Log & Post-Mortem',
        description: 'Track outages, incident severity, resolution timelines, and action items.',
        icon: '🚨',
        columns: [
          col('Incident', 'text'),
          col('Severity', 'single-select', PRIORITY),
          col('Status', 'single-select', sel([
            ['Active',        C.red   ],
            ['Investigating', C.orange],
            ['Mitigated',     C.yellow],
            ['Resolved',      C.green ],
          ])),
          col('Started At', 'datetime'),
          col('Resolved At', 'datetime'),
          col('Incident Lead', 'text'),
          col('Root Cause', 'long-text'),
          col('Action Items', 'checklist'),
          col('Post-Mortem', 'markdown'),
        ],
        views: [
          view('Incident Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'user-interviews',
        name: 'User Interviews',
        description: 'Log customer discovery calls, sentiment, and feature desires.',
        icon: '🎙️',
        columns: [
          col('Participant', 'text'),
          col('Company / Org', 'text'),
          col('Date', 'date'),
          col('Interviewer', 'text'),
          col('Sentiment', 'rating'),
          col('Recording Link', 'url'),
          col('Topics & Desires', 'multi-select', sel([
            ['UX / Usability',   C.purple],
            ['Performance',      C.yellow],
            ['Pricing',          C.blue  ],
            ['Integrations',     C.teal  ],
            ['Missing Features', C.red   ],
          ])),
          col('Key Takeaways', 'markdown'),
        ],
      },
      {
        id: 'launch-checklist',
        name: 'Launch Checklist',
        description: 'Pre-launch tasks, ship criteria, and go-live sanity checks.',
        icon: '🚀',
        columns: [
          col('Item', 'text'),
          col('Area', 'single-select', sel([
            ['Engineering',      C.blue  ],
            ['Design / UX',      C.purple],
            ['Marketing & Docs', C.teal  ],
            ['Infra / Ops',      C.orange],
          ])),
          col('Status', 'single-select', sel([
            ['Todo',        C.gray  ],
            ['In Progress', C.blue  ],
            ['Ready',       C.green ],
            ['Blocked',     C.red   ],
          ])),
          col('Owner', 'text'),
          col('Priority', 'single-select', PRIORITY),
          col('Due Date', 'date'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Launch Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'sales-pipeline',
        name: 'Sales Pipeline',
        description: 'Lightweight deal pipeline with deal values and closing dates.',
        icon: '💰',
        columns: [
          col('Deal / Account', 'text'),
          col('Contact', 'text'),
          col('Email', 'email'),
          col('Stage', 'single-select', sel([
            ['Lead',          C.gray  ],
            ['Contacted',     C.blue  ],
            ['Demo / Pitch',  C.purple],
            ['Proposal Sent', C.orange],
            ['Won',           C.green ],
            ['Lost',          C.red   ],
          ])),
          col('Deal Value', 'currency'),
          col('Target Close', 'date'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Deal Pipeline', 'kanban', { xCol: 'Stage' }, true),
        ],
      },
      {
        id: 'client-projects',
        name: 'Client Projects & Billing',
        description: 'Track client deliverables, billing status, contracts, and deadlines.',
        icon: '💼',
        columns: [
          col('Project', 'text'),
          col('Client', 'text'),
          col('Status', 'single-select', sel([
            ['Scoping',   C.gray  ],
            ['Active',    C.blue  ],
            ['Review',    C.orange],
            ['Delivered', C.teal  ],
            ['Completed', C.green ],
          ])),
          col('Billing Status', 'single-select', sel([
            ['Unbilled', C.gray  ],
            ['Invoiced', C.orange],
            ['Paid',     C.green ],
            ['Overdue',  C.red   ],
          ])),
          col('Contract Value', 'currency'),
          col('Deadline', 'date'),
          col('Deliverables', 'checklist'),
          col('Brief', 'markdown'),
        ],
        views: [
          view('Project Board', 'kanban', { xCol: 'Status' }, true),
          view('Schedule', 'timeline', { dateCol: 'Deadline' }),
        ],
      },
    ],
  },
  {
    label: 'Enterprise & Strategy',
    templates: [
      {
        id: 'initiatives',
        name: 'Initiatives',
        description: 'Capture strategic initiatives: what, why, who, and when.',
        icon: '🧭',
        columns: [
          col('Name', 'text'),
          col('Status', 'single-select', sel([
            ['Idea',     C.gray  ],
            ['Proposed', C.blue  ],
            ['Approved', C.teal  ],
            ['Active',   C.orange],
            ['Done',     C.green ],
            ['Dropped',  C.red   ],
          ])),
          col('Impact', 'single-select', sel([
            ['High',   C.green ],
            ['Medium', C.yellow],
            ['Low',    C.gray  ],
          ])),
          col('Effort', 'single-select', sel([
            ['S',  C.teal  ],
            ['M',  C.blue  ],
            ['L',  C.orange],
            ['XL', C.red   ],
          ])),
          col('Owner', 'text'),
          col('Area', 'single-select', sel([
            ['Product',       C.blue  ],
            ['Engineering',   C.teal  ],
            ['Data',          C.purple],
            ['Design',        C.pink  ],
            ['Operations',    C.orange],
          ])),
          col('Strategic Goals', 'multi-select', sel([
            ['Growth',              C.green ],
            ['Retention',           C.teal  ],
            ['Cost Reduction',      C.blue  ],
            ['Quality',             C.purple],
            ['Innovation',          C.orange],
            ['Compliance',          C.brown ],
            ['Customer Experience', C.pink  ],
          ])),
          col('Start Date', 'date'),
          col('Target Date', 'date'),
          col('Tags', 'multi-select', sel([
            ['Quick Win', C.green ],
            ['Bet',       C.orange],
            ['Platform',  C.blue  ],
            ['Debt',      C.gray  ],
          ])),
          col('Goal', 'markdown'),
        ],
        views: [
          view('Status Board', 'kanban', { xCol: 'Status' }),
        ],
      },
      {
        id: 'okr-objectives',
        name: 'Objectives',
        description: 'Define quarterly strategic objectives — the "what" of your OKR cycle.',
        icon: '🎯',
        columns: [
          col('Objective', 'text'),
          col('Owner', 'text'),
          col('Quarter', 'single-select', sel([
            ['Q1', C.teal], ['Q2', C.blue], ['Q3', C.orange], ['Q4', C.purple],
          ])),
          col('Year', 'number'),
          col('Status', 'single-select', sel([
            ['Draft',      C.gray  ],
            ['Active',     C.blue  ],
            ['Achieved',   C.green ],
            ['Missed',     C.red   ],
            ['Cancelled',  C.brown ],
          ])),
          col('Progress', 'percent'),
          col('Key Results', 'checklist'),
          col('Description', 'long-text'),
        ],
        views: [
          view('Status Board', 'kanban', { xCol: 'Status' }),
        ],
      },
      {
        id: 'okr-key-results',
        name: 'Key Results',
        description: 'Track measurable outcomes that define success for each objective.',
        icon: '📈',
        columns: [
          col('Key Result', 'text'),
          col('Objective', 'text'),
          col('Owner', 'text'),
          col('Status', 'single-select', sel([
            ['Not Started', C.gray  ],
            ['On Track',    C.green ],
            ['At Risk',     C.orange],
            ['Behind',      C.red   ],
            ['Done',        C.teal  ],
          ])),
          col('Start Value',   'number'),
          col('Current Value', 'number'),
          col('Target Value',  'number'),
          col('Unit', 'text'),
          col('Due Date', 'date'),
          col('Initiatives', 'checklist'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Status Board', 'kanban', { xCol: 'Status' }),
        ],
      },
      {
        id: 'risk-register',
        name: 'Risk Register',
        description: 'Track strategic risks by likelihood and impact, with mitigation actions and owners.',
        icon: '⚠️',
        columns: [
          col('Risk', 'text'),
          col('Category', 'single-select', sel([
            ['Strategic',    C.purple],
            ['Operational',  C.blue  ],
            ['Financial',    C.orange],
            ['Compliance',   C.brown ],
            ['Technical',    C.teal  ],
            ['People',       C.pink  ],
          ])),
          col('Likelihood', 'single-select', sel([
            ['Low',      C.green ],
            ['Medium',   C.yellow],
            ['High',     C.orange],
            ['Critical', C.red   ],
          ])),
          col('Impact', 'single-select', sel([
            ['Low',      C.green ],
            ['Medium',   C.yellow],
            ['High',     C.orange],
            ['Critical', C.red   ],
          ])),
          col('Status', 'single-select', sel([
            ['Open',       C.red   ],
            ['Mitigating', C.orange],
            ['Accepted',   C.yellow],
            ['Closed',     C.green ],
          ])),
          col('Owner', 'text'),
          col('Mitigation', 'long-text'),
          col('Review Date', 'date'),
          col('Notes', 'long-text'),
        ],
        views: [
          view('Risk Matrix', 'matrix', { xCol: 'Likelihood', yCol: 'Impact' }, true),
          view('Status Board', 'kanban', { xCol: 'Status' }),
        ],
      },
    ],
  },
  {
    label: 'Personal',
    templates: [
      {
        id: 'expense-log',
        name: 'Expense Log',
        description: 'Simple spending log for individuals or households.',
        icon: '💸',
        columns: [
          col('Date', 'date'),
          col('Description', 'text'),
          col('Category', 'single-select', sel([
            ['Housing',       C.blue  ],
            ['Food',          C.orange],
            ['Transport',     C.teal  ],
            ['Health',        C.green ],
            ['Entertainment', C.purple],
            ['Shopping',      C.pink  ],
            ['Other',         C.gray  ],
          ])),
          col('Amount', 'currency'),
          col('Account', 'single-select', sel([
            ['Cash',        C.green ],
            ['Checking',    C.blue  ],
            ['Credit Card', C.orange],
            ['Savings',     C.teal  ],
          ])),
          col('Notes', 'long-text'),
        ],
      },
      {
        id: 'recipe-book',
        name: 'Recipe Book',
        description: 'Store recipes with enough structure to filter and plan meals.',
        icon: '🍽️',
        columns: [
          col('Name', 'text'),
          col('Cuisine', 'single-select', sel([
            ['Italian',       C.green ],
            ['Asian',         C.red   ],
            ['Mexican',       C.orange],
            ['American',      C.blue  ],
            ['Mediterranean', C.teal  ],
            ['Other',         C.gray  ],
          ])),
          col('Prep Time (min)', 'number'),
          col('Servings', 'number'),
          col('Rating', 'rating'),
          col('Ingredients', 'checklist'),
          col('Photo', 'image'),
          col('Instructions', 'markdown'),
        ],
      },
      {
        id: 'reading-list',
        name: 'Reading List',
        description: 'Track books read, currently reading, and want to read.',
        icon: '📚',
        columns: [
          col('Title', 'text'),
          col('Author', 'text'),
          col('Status', 'single-select', sel([
            ['Want to Read', C.blue  ],
            ['Reading',      C.orange],
            ['Done',         C.green ],
            ['Abandoned',    C.gray  ],
          ])),
          col('Rating', 'rating'),
          col('Started', 'date'),
          col('Finished', 'date'),
          col('Cover', 'image'),
          col('Link', 'url'),
          col('Notes', 'markdown'),
        ],
        views: [
          view('Reading Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'subscriptions',
        name: 'Subscriptions',
        description: "Know what you're paying for and when it renews.",
        icon: '💳',
        columns: [
          col('Service', 'text'),
          col('Category', 'single-select', sel([
            ['Software',  C.blue  ],
            ['Streaming', C.red   ],
            ['News',      C.orange],
            ['Health',    C.green ],
            ['Utilities', C.teal  ],
            ['Other',     C.gray  ],
          ])),
          col('Cost', 'currency'),
          col('Billing Cycle', 'single-select', sel([
            ['Monthly', C.blue  ],
            ['Annual',  C.green ],
            ['Weekly',  C.orange],
          ])),
          col('Next Renewal', 'date'),
          col('Active', 'checkbox'),
          col('Manage URL', 'url'),
          col('Notes', 'long-text'),
        ],
      },
      {
        id: 'decision-journal',
        name: 'Decision Journal',
        description: 'Record why decisions were made. Revisit to close the loop.',
        icon: '🧠',
        columns: [
          col('Decision', 'text'),
          col('Decided On', 'date'),
          col('Decided By', 'text'),
          col('Status', 'single-select', sel([
            ['Pending',   C.blue ],
            ['Validated', C.green],
            ['Reversed',  C.red  ],
          ])),
          col('Expected Outcome', 'long-text'),
          col('Actual Outcome', 'long-text'),
          col('Context & Alternatives', 'markdown'),
        ],
        views: [
          view('Decisions Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'waiting-for',
        name: 'Waiting For',
        description: 'Track delegated items. Close open loops before they close you.',
        icon: '⏳',
        columns: [
          col('Item', 'text'),
          col('Delegated To', 'text'),
          col('Delegated On', 'date'),
          col('Expected By', 'date'),
          col('Status', 'single-select', sel([
            ['Waiting',   C.blue  ],
            ['Received',  C.green ],
            ['Overdue',   C.red   ],
            ['Cancelled', C.gray  ],
          ])),
          col('Follow-up Notes', 'long-text'),
        ],
        views: [
          view('Waiting Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
    ],
  },
  {
    label: 'Coding Agents',
    templates: [
      {
        id: 'agent-orchestration',
        name: 'Agent Orchestration',
        description: 'Coordinate multi-agent workflows with dependency tracking.',
        icon: '🤖',
        columns: [
          col('Task / Goal', 'text'),
          col('Agent Role', 'single-select', sel([
            ['Planner',    C.purple],
            ['Executor',   C.blue  ],
            ['Reviewer',   C.orange],
            ['Researcher', C.teal  ],
            ['Integrator', C.green ],
          ])),
          col('Assigned Agent', 'text'),
          col('Status', 'single-select', sel([
            ['Pending', C.gray  ],
            ['Running', C.blue  ],
            ['Blocked', C.red   ],
            ['Done',    C.green ],
            ['Failed',  C.brown ],
          ])),
          col('Depends On (task IDs)', 'text'),
          col('Input Artifacts', 'long-text'),
          col('Output Artifacts', 'long-text'),
          col('Failure Reason', 'long-text'),
          col('Started', 'date'),
          col('Completed', 'date'),
        ],
        views: [
          view('Agent Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'evidence-trail',
        name: 'Evidence Trail',
        description: 'Agents document decisions and reasoning for audit and handoff.',
        icon: '🔍',
        columns: [
          col('Claim / Decision', 'text'),
          col('Evidence', 'long-text'),
          col('Alternatives Rejected', 'long-text'),
          col('References', 'long-text'),
          col('Confidence', 'single-select', sel([
            ['High',      C.green ],
            ['Medium',    C.yellow],
            ['Low',       C.orange],
            ['Uncertain', C.gray  ],
          ])),
          col('Agent', 'text'),
          col('Task Context', 'text'),
          col('Recorded On', 'date'),
          col('Status', 'single-select', sel([
            ['Active',      C.green ],
            ['Superseded',  C.orange],
            ['Invalidated', C.red   ],
          ])),
        ],
        views: [
          view('Evidence Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
      {
        id: 'task-memory-graph',
        name: 'Task Memory Graph',
        description: 'Dependency-aware work graph for long-horizon agent tasks.',
        icon: '🕸️',
        columns: [
          col('Title', 'text'),
          col('Type', 'single-select', sel([
            ['Goal',       C.purple],
            ['Task',       C.blue  ],
            ['Finding',    C.teal  ],
            ['Constraint', C.orange],
            ['Decision',   C.yellow],
            ['Question',   C.gray  ],
            ['Blocker',    C.red   ],
          ])),
          col('Status', 'single-select', sel([
            ['Open',        C.blue  ],
            ['In Progress', C.orange],
            ['Done',        C.green ],
            ['Blocked',     C.red   ],
            ['Deferred',    C.gray  ],
          ])),
          col('Description', 'long-text'),
          col('Depends On (row IDs)', 'text'),
          col('Blocks (row IDs)', 'text'),
          col('Context Snapshot', 'long-text'),
          col('Priority', 'single-select', PRIORITY),
          col('Agent', 'text'),
          col('Created', 'date'),
          col('Resolved', 'date'),
          col('Resolution Notes', 'long-text'),
        ],
        views: [
          view('Graph Tasks Board', 'kanban', { xCol: 'Status' }, true),
        ],
      },
    ],
  },
]

export const ALL_TEMPLATES = TEMPLATE_GROUPS.flatMap(g => g.templates)

/**
 * Instantiates columns and preset views for a newly created table.
 *
 * @param {object} api - Degubase API client
 * @param {string} workspaceCode
 * @param {object} newTable - Table record returned by api.createTable
 * @param {object} template - Template definition from ALL_TEMPLATES
 * @returns {Promise<object|null>} The preferred view to navigate to, if any
 */
export async function instantiateTemplate(api, workspaceCode, newTable, template) {
  const colMap = new Map()
  if (template?.columns?.length) {
    for (const colDef of template.columns) {
      const createdCol = await api.createColumn(workspaceCode, newTable.code, colDef)
      if (createdCol?.code) {
        colMap.set(colDef.name.toLowerCase(), createdCol.code)
      }
    }
  }

  let targetView = newTable.views?.find(v => v.id === newTable.default_view_id) ?? null

  if (template?.views?.length) {
    for (const viewDef of template.views) {
      const config = { ...(viewDef.config || {}) }
      if (config.xCol && colMap.has(config.xCol.toLowerCase())) {
        config.xCol = colMap.get(config.xCol.toLowerCase())
      }
      if (config.yCol && colMap.has(config.yCol.toLowerCase())) {
        config.yCol = colMap.get(config.yCol.toLowerCase())
      }
      if (config.dateCol && colMap.has(config.dateCol.toLowerCase())) {
        config.dateCol = colMap.get(config.dateCol.toLowerCase())
      }

      const createdView = await api.createView(workspaceCode, newTable.code, {
        name: viewDef.name,
        type: viewDef.type,
        config,
      })

      if (viewDef.isDefault && createdView) {
        targetView = createdView
        try {
          await api.updateTable(workspaceCode, newTable.code, {
            name: newTable.name,
            context: newTable.context || '',
            icon: newTable.icon || '',
            default_view_id: createdView.id,
          })
        } catch (_) {
          // If updateTable encounters an issue, targetView navigation still succeeds
        }
      }
    }
  }

  return targetView
}
