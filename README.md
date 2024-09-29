# worklog-app
A new worklog for my work

This version of the app a gui using bubbletea 
It is a local based versions with a local database (sqlite)


### Currently working on
- ~~Make an edit view for each time entry~~
- ~~Fixing bugs in list view of entries~~
- ~~Add more validation on new entries.~~
- ~~Summary Page for Days, easily show work completed.~~
- Scoro API integration
- 

### Known Bugs
- ~~If deleting an entry list will not update until navigating away and back~~
- ~~Fix formatting limitations for hours field.~~
- fix bug with infinite scroll and more than 10 items in db.
- ~~Add save functionality for modify entry.~~ 
- Add ability to switch filter item (proj, desc).
- ~~Fix issue of modify values carrying over to next modify input.~~
- ~~Look into issue with list and new page showing stale info.~~
- ~~Create a summary view that will make it easy to copy entries to scoro.~~
- ~~Summary view will either be a list of the weeks entries, or whatever is currently in the get list. Probably the week.~~
    - Future add ability to select a week to view.
- ~~Make summary view~~
- Format Summary view nicely
- ~~Allow importing and exporting of worklog txt file for storage and seeding.~~
    - ~~Currently only looks for files that have the same syntax as old worklog script.~~
    - Improvement to above would be a file picker
- ~~Allow export of entries in data base to text file.~~
    - ~~Same format as old worklog for compatibility.~~
    - Check for common worklog syntax issues, resolve for importing.
- 

# How to use the worklog

## Summary View
### How to enter summary view?
Press ctrl + p to enter summary view. 

## Modify View
### How to enter modify view?
Press enter on an item in the list view. 

## List View
### How to enter list view?
Tab from New or Modify view to get to the list view.

## New View 
### How to enter new view?
Default view when the app starts