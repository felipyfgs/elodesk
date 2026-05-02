## ADDED Requirements

### Requirement: Contact form fields SHALL be shared via base component
The shared form fields between `EditForm.vue` and `AddModal.vue` (name, email, phone, country, avatar, custom attributes) SHALL be extracted into a `ContactFormFields.vue` base component. Both parent components SHALL use this base component via slots for their differing elements (submit button, modal wrapper, title).

#### Scenario: AddModal uses ContactFormFields
- **WHEN** the add contact modal is rendered
- **THEN** form fields SHALL be rendered by `ContactFormFields.vue`
- **AND** the modal wrapper (UModal) and submit action SHALL remain in `AddModal.vue`

#### Scenario: EditForm uses ContactFormFields
- **WHEN** the contact edit form is rendered
- **THEN** form fields SHALL be rendered by `ContactFormFields.vue`
- **AND** the page-level layout and submit action SHALL remain in `EditForm.vue`

#### Scenario: Contact creation and editing behavior is unchanged
- **WHEN** a user creates or edits a contact
- **THEN** validation, submission, and error handling SHALL behave identically to before the refactor
