/**
 * @typedef {Object} Person
 * @property {string} name - Имя человека
 * @property {string} surname - Фамилия человека
 * @property {string} [patronymic] - Отчество человека (опционально)
 * @property {string} address - Адрес человека
 * @property {string} passport - Серия и номер паспорта человека
 */

/**
 * @type {Array<Person>}
 */
const peopleData = [
  {
    name: "Петр",
    surname: "Петров",
    patronymic: "Петрович",
    address: "г. Санкт-Петербург, пр. Невский, д. 10",
    passport: "4012 345678",
  },

  {
    name: "Сергей",
    surname: "Сидоров",
    patronymic: "Сергеевич",
    address: "г. Екатеринбург, ул. Свердлова, д. 25",
    passport: "6543 210987",
  },

  {
    name: "Анна",
    surname: "Смирнова",
    patronymic: "Андреевна",
    address: "г. Новосибирск, ул. Гоголя, д. 15",
    passport: "9876 543210",
  },

  {
    name: "Олег",
    surname: "Новиков",
    address: "г. Ростов-на-Дону, ул. Советская, д. 12",
    passport: "1234 567890",
  },

  {
    name: "Татьяна",
    surname: "Морозова",
    address: "г. Самара, ул. Ленинградская, д. 5",
    passport: "5678 901234",
  },

  {
    name: "Александр",
    surname: "Волков",
    address: "г. Омск, пр. Мира, д. 18",
    passport: "9012 345678",
  },

  {
    name: "Марина",
    surname: "Соколова",
    address: "г. Челябинск, ул. Труда, д. 7",
    passport: "3456 789012",
  },

  {
    name: "Иван",
    surname: "Иванов",
    patronymic: "Иванович",
    address: "г. Москва, ул. Тверская, д. 5",
    passport: "4321 123456",
  },

  {
    name: "Мария",
    surname: "Кузнецова",
    patronymic: "Александровна",
    address: "г. Казань, ул. Кремлевская, д. 20",
    passport: "8765 432109",
  },

  {
    name: "Алексей",
    surname: "Николаев",
    address: "г. Уфа, ул. Комсомольская, д. 30",
    passport: "7890 123456",
  },

  {
    name: "Екатерина",
    surname: "Федорова",
    patronymic: "Владимировна",
    address: "г. Пермь, ул. Ленина, д. 8",
    passport: "5678 091234",
  },

  {
    name: "Дмитрий",
    surname: "Орлов",
    address: "г. Новосибирск, ул. Советская, д. 15",
    passport: "5678 789012",
  },
];

export default {
  peopleData,
};
