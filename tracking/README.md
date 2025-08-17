# Development Tracking

This directory contains category-based tracking files for team visibility and development continuity.

## Purpose

These files provide persistent tracking across development sessions and team members, ensuring:
- **Visibility**: Current work status across all categories
- **Handoffs**: Easy transitions between developers
- **Planning**: Historical context for future work
- **Reviews**: Complete picture of changes

## Categories

- **`features.md`** - New functionality, enhancements, capabilities
- **`bugfixes.md`** - Bug resolutions, corrections, patches  
- **`refactoring.md`** - Code improvements without behavior changes
- **`documentation.md`** - Docs updates, README changes
- **`infrastructure.md`** - Build, CI/CD, deployment changes
- **`performance.md`** - Optimization work, performance improvements

## Usage

### Starting Work
1. Choose the appropriate category file
2. Add a new entry using the template format
3. Set status to "In Progress"
4. Add initial task list

### During Work
1. Update task checkboxes as you complete work
2. Add files to the "Files Modified" section
3. Document important decisions in "Notes"

### Completing Work
1. Mark all tasks as complete
2. Change status to "Completed"
3. Add final summary of what was accomplished

### Entry Format

```markdown
## [YYYY-MM-DD] [Brief Description]

**Status**: [In Progress | Completed | Cancelled]  
**Developer**: [Name/Username]  
**Related Issues**: [#123, #456]  

### Summary
Brief description of the work being done.

### Tasks
- [x] Task 1 description
- [x] Task 2 description  
- [ ] Task 3 description

### Files Modified
- `path/to/file1.go` - Description of changes
- `path/to/file2.go` - Description of changes

### Notes
Any important decisions, blockers, or context for future developers.

---
```

## Best Practices

- **Be specific**: Use descriptive titles and detailed task descriptions
- **Update frequently**: Keep files current as work progresses
- **Link issues**: Reference GitHub issues when applicable
- **Document decisions**: Record why certain approaches were chosen
- **Team awareness**: Check existing entries before starting similar work

## Integration with Todo Tools

This system complements the `todo_write` tool usage:
- Use `todo_write` for immediate session planning and tracking
- Use tracking files for persistent, team-visible documentation
- Update both as work progresses for maximum visibility
