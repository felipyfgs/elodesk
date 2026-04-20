import { useAgentsStore } from '~/stores/agents'
import { useInboxesStore } from '~/stores/inboxes'
import { useTeamsStore } from '~/stores/teams'

export type FilterAttributeType
  = | 'enum'
    | 'text'
    | 'number'
    | 'date'
    | 'agent'
    | 'team'
    | 'inbox'
    | 'bool'

export type FilterOperator
  = | 'equal_to'
    | 'not_equal_to'
    | 'contains'
    | 'starts_with'
    | 'greater_than'
    | 'less_than'
    | 'in'
    | 'between'
    | 'is_null'
    | 'is_not_null'

export interface FilterEnumOption {
  value: string | number
  label: string
}

export interface FilterAttribute {
  key: string
  label: string
  type: FilterAttributeType
  options?: FilterEnumOption[]
}

export const OPERATORS_BY_TYPE: Record<FilterAttributeType, FilterOperator[]> = {
  enum: ['equal_to', 'not_equal_to', 'in'],
  text: ['equal_to', 'not_equal_to', 'contains', 'starts_with', 'is_null', 'is_not_null'],
  number: ['equal_to', 'not_equal_to', 'greater_than', 'less_than', 'between', 'is_null', 'is_not_null'],
  date: ['equal_to', 'greater_than', 'less_than', 'between'],
  agent: ['equal_to', 'not_equal_to', 'in', 'is_null', 'is_not_null'],
  team: ['equal_to', 'not_equal_to', 'in', 'is_null', 'is_not_null'],
  inbox: ['equal_to', 'not_equal_to', 'in'],
  bool: ['equal_to', 'not_equal_to']
}

export const OPERATORS_NO_INPUT = new Set<FilterOperator>(['is_null', 'is_not_null'])
export const OPERATORS_MULTI_INPUT = new Set<FilterOperator>(['in', 'between'])

export function useFilterAttributes() {
  const { t } = useI18n()
  const agentsStore = useAgentsStore()
  const inboxesStore = useInboxesStore()
  const teamsStore = useTeamsStore()

  const conversationAttributes = computed<FilterAttribute[]>(() => [
    {
      key: 'status',
      label: t('savedFilters.attributes.status'),
      type: 'enum',
      options: [
        { value: 0, label: t('conversations.status.open') },
        { value: 2, label: t('conversations.status.pending') },
        { value: 3, label: t('conversations.status.snoozed') },
        { value: 1, label: t('conversations.status.resolved') }
      ]
    },
    {
      key: 'assignee_id',
      label: t('savedFilters.attributes.assignee'),
      type: 'agent',
      options: agentsStore.items.map(a => ({ value: a.id, label: a.name || a.email }))
    },
    {
      key: 'inbox_id',
      label: t('savedFilters.attributes.inbox'),
      type: 'inbox',
      options: inboxesStore.list.map(i => ({ value: Number(i.id), label: i.name }))
    },
    {
      key: 'team_id',
      label: t('savedFilters.attributes.team'),
      type: 'team',
      options: teamsStore.list.map(tm => ({ value: Number(tm.id), label: tm.name }))
    },
    {
      key: 'created_at',
      label: t('savedFilters.attributes.createdAt'),
      type: 'date'
    },
    {
      key: 'last_activity_at',
      label: t('savedFilters.attributes.lastActivity'),
      type: 'date'
    }
  ])

  const contactAttributes = computed<FilterAttribute[]>(() => [
    { key: 'name', label: t('savedFilters.attributes.name'), type: 'text' },
    { key: 'email', label: t('savedFilters.attributes.email'), type: 'text' },
    { key: 'phone_number', label: t('savedFilters.attributes.phone'), type: 'text' },
    { key: 'identifier', label: t('savedFilters.attributes.identifier'), type: 'text' },
    { key: 'blocked', label: t('savedFilters.attributes.blocked'), type: 'bool', options: [
      { value: 'true', label: t('common.yes') },
      { value: 'false', label: t('common.no') }
    ] },
    { key: 'created_at', label: t('savedFilters.attributes.createdAt'), type: 'date' },
    { key: 'last_activity_at', label: t('savedFilters.attributes.lastActivity'), type: 'date' }
  ])

  function attributesFor(filterType: 'conversation' | 'contact'): ComputedRef<FilterAttribute[]> {
    return filterType === 'conversation' ? conversationAttributes : contactAttributes
  }

  function operatorLabel(op: FilterOperator): string {
    return t(`savedFilters.operators.${op}`)
  }

  function operatorsFor(type: FilterAttributeType): { value: FilterOperator, label: string }[] {
    return OPERATORS_BY_TYPE[type].map(op => ({ value: op, label: operatorLabel(op) }))
  }

  return {
    conversationAttributes,
    contactAttributes,
    attributesFor,
    operatorsFor,
    operatorLabel
  }
}
