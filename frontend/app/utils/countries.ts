import countriesData from './countries.data.json'

export interface Country {
  name: string
  dial_code: string
  emoji: string
  id: string
}

export const countries: Country[] = countriesData as Country[]

export default countries
