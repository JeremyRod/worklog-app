# worklog-app
A new worklog for my work

This version of the app a gui using bubbletea 
It is a local based versions with a local database (sqlite)

# How to use the worklog
Navigate the worklog with arrow keys for entry field and tab/enter to interact/navigate with pages.

Ctrl+C on any page will close the app so make sure to save all data before closing.

## New View 
### How to enter new view?
Default view when the app starts

### What to do in New View?
This is where new entrys are created and notes are added.
Hit the Save button to add the entry to the database
**Tab** will move between the New and List View when continually pressed 

## List View
### How to enter list view?
**Tab** from New or Modify view to get to the list view.
**Tab** will move between the New and List View when continually pressed

### What to do in list view 
You can view all the current items that have been added to the DB. The list view will start with the latest 10 items and infinite scrol until the last item is reached

## Summary View
### How to enter summary view?
Press ctrl + p to enter summary view when in the list view. 
Select the dates you want to view and then press ctrl + p again.

### What to do in summary view
Here you can see all entries from the last 7 days.
If all these entries seem correct you can hit **Enter** to start the upload process. 
First you will be prompted to login and then link any project codes to scoro tasks that are currently unlinked.
Once complete hit **Enter** again from the summary page to upload all entries accumulated from the week.

## Modify View
### How to enter modify view?
Press enter on an item in the list view. 

### What to do in modify view?
In this view you are able to edit an entry and then save it or delete it if no longer required. 
You are also able to upload a single entry or unlink it if the scoro task bucket has changed and needs to be updated. 
***When a time entry is submitted to an old task, the consequences are untested an unknown. This could range from a http failure response to a success with a new entry being lost to the void. Please check all data is accurate and appropriately organised before submitting. If an issue occurs contact your scoro admin***

## Login 
### Why login? 
The login form takes in the users Scoro username and password and uses this to get a user_token, this is to avoid needing the api_key which some businesses might not want to provide to workers. 

The login data is not saved at all and is only used to obtain auth. The user will need to login on every new session and may need to re-enter details if auth is considered stale in a long session. 

If you do not want to enter login details, a `user.env` file can be used to store these details for the worklog app to read from when required. 

```
SCOROUSER=jeremy.rodarellis@boostdesign.com.au
SCOROPASSWORD=abc123
```

If this doesn't exist then the user will need to use the login form when prompted to upload

## Notes
### What are these used for 
Some managers would prefer a short recap of the action taken in a time entry period for reporting. Some users want to document important notes and task issues/fixes. The notes portions will bridge that gap, the notes are a personal optional note taking field associated with the entry that will be saved but will not be uploaded with the entry when uploading to scoro, this will continue to be the description.


User presses upload on an entry. If the event_id is unknown, the user will be prompted to select a Project/Event for all future project code (currently only for the current instance)
This view should first prompt the user to enter their username and passwrod for scoro in to get a user token, another option can be using an env file to get the required details.

If the Scoro task/bucket has changed after a Project code has been linked, the user can unlink and relink to a new task.

### Currently working on
- ~~Make an edit view for each time entry~~
- ~~Fixing bugs in list view of entries~~
- ~~Add more validation on new entries.~~
- ~~Summary Page for Days, easily show work completed.~~
- ~~Scoro API integration~~
    - ~~Allow Submitting of Single entry~~
    - ~~Provide form for user provided details to retreive user token.~~
- ~~Save map of projcode to event_id to database~~
- ~~error logging~~
- ~~delete links~~
- ~~checking of scoro error responses~~ 
- and handling them
- ~~Add a new view that prompts for scoro login details.~~
    - ~~Once user token is obtained for the session, the user will then need to link project codes to scoro buckets. ~~
    - ~~Save in a map or in another table in the DB.~~
- ~~Add versioning~~
- ~~Export worklog~~
- ~~Notes view~~
- ~~Summary view revamp, format text a little better, change hours view to be hh:mm.~~ 
- ~~Allow saving of a weeks entries from summary view.~~
- ~~Add ability to select dates for summary.~~
- ~~Ability to get summary of current day~~
- ~~Total days hours in summary view~~
- ~~Add ability to leave summary upload if wanted.~~
- ~~Add activity type to uploaded time entry.~~
- ~~Add ability to skip upload of certain proj codes.~~

### Future Additions 
- error log file rotations?
- Filepicker for importing.
- Add inbuilt search functionality based on any input
- Add testing to code

### Known Bugs
- ~~If deleting an entry list will not update until navigating away and back~~
- ~~Fix formatting limitations for hours field.~~
- ~~fix bug with infinite scroll and more than 10 items in db.~~
- ~~Add save functionality for modify entry.~~ 
- ~~Fix issue of modify values carrying over to next modify input.~~
- ~~Look into issue with list and new page showing stale info.~~
- ~~Create a summary view that will make it easy to copy entries to scoro.~~
- ~~Summary view will either be a list of the weeks entries, or whatever is currently in the get list. Probably the week.~~
- ~~Make summary view~~
- ~~Allow importing and exporting of worklog txt file for storage and seeding.~~
    - ~~Currently only looks for files that have the same syntax as old worklog script.~~
    - Improvement to above would be a file picker
- ~~Allow export of entries in data base to text file.~~
    - ~~Same format as old worklog for compatibility.~~
- ~~Going to modify view, then tabbing back to list view hides items~~
- ~~Going to modify view then going back when past the first 10 items will index incorrectly~~
- ~~Going into modify view and back will forget the current list index.~~
- Resizing terminal window in summary view will cause a summary artifact to cover the screen making it unusable, requires restart.
- ~~Getting next 10 list items duplicates the edge value.~~
- ~~Unlink will remove db entry but not update the map in code.~~
- ~~After pressing upload and being asked to link to a task, the mod state will be reset even if the upload was not complete.~~
- ~~exporting broken due to notes field.~~
