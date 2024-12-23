# addtocalendars

This package is useful for those developers who are creating email templates and using GO as backend language.
By using this package you can integrate Add To Calendar button in your email and redirect to calendar url which is obtained by this package.

Currently this package can only generate google calendar url but this package is in development for other calendars.

You can give Add To Calendar button in your email html template and in href property of that button you can directly pass the url generated by this package.
## Tech Stack

**Language:** Golang


## Installation

Install addtocalendar with go get

```bash
  go get -u github.com/inayatmemon/addtocalendar@latest
```
or
```bash
  go get -u github.com/inayatmemon/addtocalendar@v1.0.2
```
    
## Documentation

Initialize the variable with AddToCalendar struct of this package. And initialize the fields of this package with your data as example shown below.

```javascript
a := addtocalendars.AddToCalendar{
    Title:                   "My teaddtocalendars", // Title of the meeting
    Details:                 "Test details", // details of the meeting
    Location:                "Mumbai", // location of the meeting
    Timezone:                "Asia/Kolkata", // timezone in which meeting is going to happen
    EventStartUnixTimestamp: time.Now().Unix(), // start time of the meeting
    EventEndUnixTimestamp:   time.Now().Unix() + 3600, // end time of the meeting
}
```

After the initialization just call the method for this struct or data as shown below.

```javascript
eventUrl, err := a.AddToCalendar()
if err != nil {
    fmt.Println("err in add to calendar: ", err)
    return
}
```

This method will return two variables,
1) **eventUrl**: Actual calendar url which you can redirect to browser and it will open the calendar page with the data you have passed.
2) **err**: Error (If the url could not generate due to some runtime error then you can handle with this variable).

## Authors

- [@inayatmemon](https://www.github.com/inayatmemon)


## 🚀 About Me
I'm a full stack developer...


## Contributing

Contributions are always welcome!


## 🛠 Skills
Goalng, mongodb, redis, rabbitmq, websocket, elasticsearch..


## License

[MIT](https://choosealicense.com/licenses/mit/)

