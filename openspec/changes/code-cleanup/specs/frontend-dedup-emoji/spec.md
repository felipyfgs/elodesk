## ADDED Requirements

### Requirement: Emoji picker logic SHALL be shared via composable
The emoji picker logic (import, `EmojiPickerSelectEvent` interface, theme computed, group names computed, `onEmojiSelect` handler) SHALL be extracted into a single `useEmojiPicker` composable used by both `ComposerToolbar.vue` and `SendMessageEmojiButton.vue`.

#### Scenario: ComposerToolbar uses useEmojiPicker
- **WHEN** the composer toolbar renders the emoji button
- **THEN** emoji picker state and handlers SHALL be provided by `useEmojiPicker()`
- **AND** no duplicate `EmojiPickerSelectEvent` interface, `emojiTheme`, or `emojiGroupNames` definitions SHALL exist in the component

#### Scenario: SendMessageEmojiButton uses useEmojiPicker
- **WHEN** the contact send message dialog renders the emoji button
- **THEN** emoji picker state and handlers SHALL be provided by `useEmojiPicker()`
- **AND** the component SHALL not contain its own copy of emoji picker logic

#### Scenario: Emoji selection behavior is unchanged
- **WHEN** a user selects an emoji from the picker in either component
- **THEN** the emoji SHALL be inserted into the text input at cursor position
- **AND** the behavior SHALL be identical to before the refactor
